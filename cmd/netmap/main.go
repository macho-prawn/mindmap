package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"netmap/internal/app"
	"netmap/internal/provider"
	"netmap/internal/version"

	googleoauth "golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var newComputeProvider = func(ctx context.Context) (provider.DiscoveryProvider, error) {
	return provider.NewComputeProvider(ctx)
}

var adcPreflight = func(ctx context.Context) error {
	creds, err := googleoauth.FindDefaultCredentials(ctx, compute.CloudPlatformScope)
	if err != nil {
		return err
	}
	if creds == nil || creds.TokenSource == nil {
		return errors.New("application default credentials are missing a token source")
	}
	if _, err := creds.TokenSource.Token(); err != nil {
		return err
	}
	return nil
}

const (
	adcGuidanceLine    = "There seem to be ADC-related authenticaton issues encountered. Please run the following command;"
	adcGuidanceCommand = "$ gcloud auth login --update-adc --force"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "version" {
		if len(args) != 1 {
			fmt.Fprintln(stderr, "version command does not accept additional arguments")
			return 1
		}
		fmt.Fprintln(stdout, version.Value)
		return 0
	}

	files := app.RealFileStore{}

	opts, err := app.ParseOptions(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if opts.ShowHelp {
		fmt.Fprint(stdout, opts.Usage)
		return 0
	}

	if err := adcPreflight(ctx); err != nil {
		fmt.Fprintln(stderr, normalizeCredentialError(err))
		return 1
	}

	input, err := app.Validate(files, args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if input.Options.ShowHelp {
		fmt.Fprint(stdout, input.Options.Usage)
		return 0
	}

	discovery, err := newComputeProvider(ctx)
	if err != nil {
		fmt.Fprintln(stderr, normalizeCredentialError(err))
		return 1
	}

	cli, err := app.New(
		files,
		discovery,
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := cli.RunValidated(ctx, input); err != nil {
		fmt.Fprintln(stderr, normalizeCredentialError(err))
		return 1
	}
	return 0
}

func normalizeCredentialError(err error) error {
	if !isCredentialError(err) {
		return err
	}
	return fmt.Errorf("%s\n\n%s\n\nUnderlying error: %v", adcGuidanceLine, adcGuidanceCommand, err)
}

func isCredentialError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		if apiErr.Code == 401 {
			return true
		}

		message := strings.ToLower(strings.TrimSpace(apiErr.Message))
		if strings.Contains(message, "invalid authentication credentials") ||
			strings.Contains(message, "authentication credential") ||
			strings.Contains(message, "unauthenticated") ||
			strings.Contains(message, "insufficient authentication scopes") {
			return true
		}
	}

	message := strings.ToLower(err.Error())
	patterns := []string{
		"could not find default credentials",
		"application default credentials",
		"default credentials",
		"oauth2/google: error getting credentials",
		"oauth2: cannot fetch token",
		"invalid_grant",
		"unable to generate access token",
		"request had invalid authentication credentials",
		"request is missing required authentication credential",
		"unauthenticated",
		"insufficient authentication scopes",
		"anonymous credentials cannot be refreshed",
	}
	for _, pattern := range patterns {
		if strings.Contains(message, pattern) {
			return true
		}
	}
	return false
}
