package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/spf13/cobra"
	"github.com/unkeyed/unkey/apps/agent/pkg/connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/env"
	"github.com/unkeyed/unkey/apps/agent/pkg/heartbeat"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/ratelimit"
)

var (
	envFile                string
	enableRatelimitService bool
)

func init() {
	rootCmd.AddCommand(AgentCmd)

	AgentCmd.Flags().StringVarP(&envFile, "env", "e", "", "specify the .env file path (by default no .env file is loaded)")
	AgentCmd.Flags().BoolVar(&enableRatelimitService, "enable-ratelimit", false, "enable the ratelimit service")
}

// AgentCmd represents the agent command
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "A brief description of your command",

	Run: func(cmd *cobra.Command, args []string) {
		if envFile != "" {

			err := godotenv.Load(envFile)
			if err != nil {
				log.Fatal("Error loading .env file")
			}
		}

		e := env.Env{
			ErrorHandler: func(err error) {
				log.Fatalf("unable to load environment variable: %s", err.Error())
			},
		}

		logConfig := &logging.Config{
			Debug:  e.Bool("DEBUG", false),
			Writer: []io.Writer{},
		}
		axiomToken := e.String("AXIOM_TOKEN", "")
		axiomOrgId := e.String("AXIOM_ORG_ID", "")
		if axiomToken != "" && axiomOrgId != "" {
			axiomWriter, err := logging.NewAxiomWriter(logging.AxiomWriterConfig{
				AxiomToken: axiomToken,
				AxiomOrgId: axiomOrgId,
			})
			if err != nil {
				log.Fatalf("unable to create axiom writer: %s", err)
			}
			logConfig.Writer = append(logConfig.Writer, axiomWriter)
		}

		logger := logging.New(logConfig)
		logger.Info().Strs("env", os.Environ()).Send()

		srv, err := connect.New(connect.Config{Logger: logger})
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create service")
		}

		if enableRatelimitService {

			rl, err := ratelimit.New(ratelimit.Config{
				Logger: logger,
			})
			if err != nil {
				logger.Fatal().Err(err).Msg("failed to create service")
			}

			srv.AddService(connect.NewRatelimitServer(rl))
			logger.Info().Msg("started ratelimit service")
		}

		heartbeatUrl := os.Getenv("HEARTBEAT_URL")
		if heartbeatUrl != "" {
			h := heartbeat.New(heartbeat.Config{
				Url:    heartbeatUrl,
				Logger: logger,
			})
			go h.Run()
		}
		err = srv.Listen(fmt.Sprintf(":%s", e.String("PORT", "8080")))
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to listen")
		}

		// run something

		cShutdown := make(chan os.Signal, 1)
		signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

		<-cShutdown

	},
}
