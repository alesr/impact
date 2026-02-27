package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/alesr/impact/internal/config"
	"github.com/alesr/impact/internal/estimate"
	"github.com/alesr/impact/internal/pkg/progress"
	"github.com/alesr/impact/internal/pkg/strx"
	"github.com/alesr/impact/internal/plan"
	"github.com/alesr/impact/internal/report"
	"github.com/alesr/impact/internal/scw/catalog"
	"github.com/alesr/impact/internal/scw/footprint"
	"github.com/alesr/impact/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	userAgent            = "impact/dev"
	terraformShowTimeout = 2 * time.Minute
)

var errUsage = errors.New("usage: impact <command> (run 'impact --help')")

type catalogProductLister interface {
	ListAllProducts(ctx context.Context) ([]catalog.Product, error)
}

func Run(args []string) error {
	rootCmd := newRootCmd()
	rootCmd.SetArgs(args)
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	return rootCmd.Execute()
}

type planOptions struct {
	planFile      string
	fromTerraform bool
	format        string
	tuiMode       bool
}

type actualOptions struct {
	org               string
	start             string
	end               string
	projects          string
	regions           string
	zones             string
	serviceCategories string
	productCategories string
	format            string
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = "impact"
	cmd.Short = "scaleway environmental footprint cli"
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		if err := cmd.Help(); err != nil {
			return err
		}
		return errUsage
	}
	cmd.AddCommand(newPlanCmd(), newActualCmd(), newDoctorCmd())
	return cmd
}

func newPlanCmd() *cobra.Command {
	opts := planOptions{}

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "estimate impact from terraform plan",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runPlan(opts)
		},
	}

	cmd.Flags().StringVar(&opts.planFile, "file", "", "terraform show -json plan file")
	cmd.Flags().BoolVar(&opts.fromTerraform, "from-terraform", false, "read terraform show -json from local terraform command")
	cmd.Flags().StringVar(&opts.format, "format", "table", "output format: table|json")
	cmd.Flags().BoolVar(&opts.tuiMode, "tui", false, "interactive terminal UI for plan report")

	return cmd
}

func newActualCmd() *cobra.Command {
	var opts actualOptions

	cmd := &cobra.Command{
		Use:   "actual",
		Short: "query measured footprint impact",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runActual(opts)
		},
	}

	cmd.Flags().StringVar(&opts.org, "org", "", "organization id (defaults to SCW_ORGANIZATION_ID)")
	cmd.Flags().StringVar(&opts.start, "start", "", "start date (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&opts.end, "end", "", "end date (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&opts.projects, "project", "", "comma-separated project IDs filter")
	cmd.Flags().StringVar(&opts.regions, "region", "", "comma-separated regions filter")
	cmd.Flags().StringVar(&opts.zones, "zone", "", "comma-separated zones filter")
	cmd.Flags().StringVar(&opts.serviceCategories, "service-category", "", "comma-separated service categories filter")
	cmd.Flags().StringVar(&opts.productCategories, "product-category", "", "comma-separated product categories filter")
	cmd.Flags().StringVar(&opts.format, "format", "table", "output format: table|json")

	return cmd
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "run diagnostics",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDoctor()
		},
	}
}

func runPlan(opts planOptions) error {
	if opts.planFile != "" && opts.fromTerraform {
		return errors.New("could not build plan report: use either --file or --from-terraform, not both")
	}

	if opts.planFile == "" && !opts.fromTerraform {
		return errors.New("could not build plan report: provide --file or --from-terraform")
	}

	if opts.tuiMode {
		return tui.RunPlanReportLoading(
			func() (estimate.Report, error) {
				return buildPlanReport(opts.planFile, opts.fromTerraform)
			},
		)
	}

	var rep estimate.Report
	if err := runWithSpinner("processing plan and fetching catalog", func() error {
		var runErr error
		rep, runErr = buildPlanReport(opts.planFile, opts.fromTerraform)
		return runErr
	}); err != nil {
		return err
	}
	return outputPlanReport(opts.format, rep)
}

func buildPlanReport(planFile string, fromTerraform bool) (estimate.Report, error) {
	var (
		changes []plan.ResourceChange
		err     error
	)

	if fromTerraform {
		changes, err = readChangesFromTerraform()
	} else {
		changes, err = plan.ParseFile(planFile)
	}

	if err != nil {
		return estimate.Report{}, err
	}

	env, err := config.LoadScalewayFromEnv()
	if err != nil {
		return estimate.Report{}, err
	}

	catalogClient, err := catalog.NewClient(
		catalog.WithBaseURL(env.APIBaseURL),
		catalog.WithUserAgent(userAgent),
		catalog.WithTimeout(15*time.Second),
	)
	if err != nil {
		return estimate.Report{}, err
	}
	return buildEstimateReport(context.Background(), changes, catalogClient)
}

func buildEstimateReport(ctx context.Context, changes []plan.ResourceChange, lister catalogProductLister) (estimate.Report, error) {
	products, err := lister.ListAllProducts(ctx)
	if err != nil {
		return estimate.Report{}, fmt.Errorf("could not fetch catalog products: %w", err)
	}
	return estimate.Build(changes, products), nil
}

func runActual(opts actualOptions) error {
	env, err := config.LoadScalewayFromEnv()
	if err != nil {
		return err
	}

	orgID := strings.TrimSpace(opts.org)
	if orgID == "" {
		orgID = env.OrganizationID
	}

	if orgID == "" {
		return errors.New("could not resolve organization id (use --org or SCW_ORGANIZATION_ID)")
	}

	var startDate *time.Time
	if strings.TrimSpace(opts.start) != "" {
		t, err := parseDate(opts.start)
		if err != nil {
			return fmt.Errorf("could not parse --start: %w", err)
		}
		startDate = &t
	}

	var endDate *time.Time
	if strings.TrimSpace(opts.end) != "" {
		t, err := parseDate(opts.end)
		if err != nil {
			return fmt.Errorf("could not parse --end: %w", err)
		}
		endDate = &t
	}

	footprintClient, err := footprint.NewClient(
		env.AccessKey,
		env.SecretKey,
		footprint.WithBaseURL(env.APIBaseURL),
		footprint.WithUserAgent(userAgent),
		footprint.WithTimeout(15*time.Second),
	)
	if err != nil {
		return err
	}

	queryReq := footprint.QueryImpactDataRequest{
		OrganizationID:    orgID,
		StartDate:         startDate,
		EndDate:           endDate,
		ProjectIDs:        strx.ParseCSV(opts.projects),
		Regions:           strx.ParseCSV(opts.regions),
		Zones:             strx.ParseCSV(opts.zones),
		ServiceCategories: nil,
		ProductCategories: nil,
	}

	serviceCategories, err := parseServiceCategories(opts.serviceCategories)
	if err != nil {
		return err
	}

	productCategories, err := parseProductCategories(opts.productCategories)
	if err != nil {
		return err
	}

	queryReq.ServiceCategories = serviceCategories
	queryReq.ProductCategories = productCategories

	var resp *footprint.QueryImpactDataResponse
	if err := runWithSpinner("querying actual footprint data", func() error {
		var runErr error
		resp, runErr = footprintClient.QueryImpactData(context.Background(), queryReq)
		return runErr
	}); err != nil {
		return err
	}
	return outputActualReport(opts.format, resp)
}

func readChangesFromTerraform() ([]plan.ResourceChange, error) {
	ctx, cancel := context.WithTimeout(context.Background(), terraformShowTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "terraform", "show", "-json")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText == "" {
			return nil, fmt.Errorf("could not run terraform show -json: %w", err)
		}
		return nil, fmt.Errorf("could not run terraform show -json: %s", stderrText)
	}
	return plan.ParseBytes(out)
}

func runDoctor() error {
	env, err := config.LoadScalewayFromEnv()
	if err != nil {
		return err
	}

	missing := make([]string, 0, 2)

	if env.AccessKey == "" {
		missing = append(missing, "SCW_ACCESS_KEY")
	}

	if env.SecretKey == "" {
		missing = append(missing, "SCW_SECRET_KEY")
	}

	if env.OrganizationID == "" {
		missing = append(missing, "SCW_ORGANIZATION_ID")
	}

	status := map[string]string{
		"api_base_url": env.APIBaseURL,
		"auth":         "ok",
	}

	if len(missing) > 0 {
		status["auth"] = "missing: " + strings.Join(missing, ",")
	}

	catalogClient, err := catalog.NewClient(
		catalog.WithBaseURL(env.APIBaseURL),
		catalog.WithTimeout(10*time.Second),
	)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := catalogClient.ListProducts(ctx, catalog.ListProductsRequest{Page: 1, PageSize: 1}); err != nil {
		status["catalog"] = "error: " + err.Error()
	} else {
		status["catalog"] = "ok"
	}

	if env.AccessKey != "" && env.SecretKey != "" && env.OrganizationID != "" {
		footprintClient, err := footprint.NewClient(
			env.AccessKey,
			env.SecretKey,
			footprint.WithBaseURL(env.APIBaseURL),
			footprint.WithTimeout(10*time.Second),
		)
		if err != nil {
			return err
		}

		startDate, endDate := doctorQueryWindow(time.Now().UTC())

		if _, err := footprintClient.QueryImpactData(
			ctx,
			footprint.QueryImpactDataRequest{
				OrganizationID: env.OrganizationID,
				StartDate:      &startDate,
				EndDate:        &endDate,
			},
		); err != nil {
			status["footprint"] = "error: " + err.Error()
		} else {
			status["footprint"] = "ok"
		}
	} else {
		status["footprint"] = "skipped (missing auth)"
	}

	keys := slices.Collect(maps.Keys(status))
	slices.Sort(keys)

	for _, k := range keys {
		fmt.Printf("%s: %s\n", k, status[k])
	}
	return nil
}

func outputPlanReport(format string, rep estimate.Report) error {
	switch normalizeFormat(format) {
	case "json":
		return report.PrintJSON(rep)
	case "table":
		return report.PrintTable(rep)
	default:
		return fmt.Errorf("could not render output format %q (use table or json)", format)
	}
}

func outputActualReport(format string, rep *footprint.QueryImpactDataResponse) error {
	switch normalizeFormat(format) {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)

	case "table":
		fmt.Printf(
			"Period: %s -> %s\nTotal kgCO2e/month: %.6f\nTotal m3 water/month: %.6f\n\n",
			rep.StartDate.Format(time.RFC3339), rep.EndDate.Format(time.RFC3339),
			rep.TotalImpact.KgCO2Equivalent, rep.TotalImpact.M3WaterUsage,
		)

		for _, project := range rep.Projects {
			fmt.Printf("Project %s: kgCO2e=%.6f m3=%.6f\n", project.ProjectID, project.TotalProjectImpact.KgCO2Equivalent, project.TotalProjectImpact.M3WaterUsage)
		}
		return nil

	default:
		return fmt.Errorf("could not render output format %q (use table or json)", format)
	}
}

const dateLayout = "2006-01-02"

func parseDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("could not parse date: date is empty")
	}

	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}

	if t, err := time.Parse(dateLayout, raw); err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("could not validate date %q (use YYYY-MM-DD or RFC3339)", raw)
}

func normalizeFormat(format string) string {
	return strings.ToLower(strings.TrimSpace(format))
}

func parseServiceCategories(raw string) ([]footprint.ServiceCategory, error) {
	values := strx.ParseCSV(raw)
	if len(values) == 0 {
		return nil, nil
	}

	out := make([]footprint.ServiceCategory, 0, len(values))
	for _, value := range values {
		switch normalizeEnumToken(value) {
		case "baremetal":
			out = append(out, footprint.ServiceCategoryBaremetal)
		case "compute":
			out = append(out, footprint.ServiceCategoryCompute)
		case "storage":
			out = append(out, footprint.ServiceCategoryStorage)
		default:
			return nil, fmt.Errorf("could not validate --service-category value %q (allowed: baremetal, compute, storage)", value)
		}
	}

	return out, nil
}

func parseProductCategories(raw string) ([]footprint.ProductCategory, error) {
	values := strx.ParseCSV(raw)
	if len(values) == 0 {
		return nil, nil
	}

	out := make([]footprint.ProductCategory, 0, len(values))
	for _, value := range values {
		switch normalizeEnumToken(value) {
		case "applesilicon":
			out = append(out, footprint.ProductCategoryAppleSilicon)
		case "blockstorage":
			out = append(out, footprint.ProductCategoryBlockStorage)
		case "dedibox":
			out = append(out, footprint.ProductCategoryDedibox)
		case "elasticmetal":
			out = append(out, footprint.ProductCategoryElasticMetal)
		case "instances":
			out = append(out, footprint.ProductCategoryInstances)
		case "objectstorage":
			out = append(out, footprint.ProductCategoryObjectStorage)
		default:
			return nil, fmt.Errorf("could not validate --product-category value %q (allowed: applesilicon, blockstorage, dedibox, elasticmetal, instances, objectstorage)", value)
		}
	}

	return out, nil
}

func normalizeEnumToken(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, "_", "")
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, " ", "")
	return v
}

func doctorQueryWindow(now time.Time) (time.Time, time.Time) {
	end := now.UTC().Truncate(time.Second)
	start := end.AddDate(0, 0, -30)
	return start, end
}

func runWithSpinner(message string, fn func() error) error {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return fn()
	}
	return progress.RunCLISpinner(os.Stderr, message, fn)
}
