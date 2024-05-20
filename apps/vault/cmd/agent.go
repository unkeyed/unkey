package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/spf13/cobra"
	"github.com/unkeyed/unkey/apps/vault/pkg/connect"
	"github.com/unkeyed/unkey/apps/vault/pkg/env"
	"github.com/unkeyed/unkey/apps/vault/pkg/logging"
	"github.com/unkeyed/unkey/apps/vault/pkg/service"
	"github.com/unkeyed/unkey/apps/vault/pkg/storage"
)


var (
	envFile string
)

func init() {
	rootCmd.AddCommand(AgentCmd)

	AgentCmd.Flags().StringVarP(&envFile, "env", "e", "","specify the .env file path (by default no .env file is loaded)")
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
			Debug: e.Bool("DEBUG", false),
		}
		logger := logging.New(logConfig)

		


		storage, err := storage.NewS3(storage.S3Config{
			S3URL:             e.String("S3_URL"),
			S3Bucket:          e.String("S3_BUCKET"),
			S3AccessKeyId:     e.String("S3_ACCESS_KEY_ID"),
			S3AccessKeySecret: e.String("S3_ACCESS_KEY_SECRET"),
			Logger:            logger,
		})
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create storage")
		}


		masterKeys := e.Strings("VAULT_MASTER_KEYS")

		vault, err := service.New(service.Config{
			Logger:  logger,
			Storage: storage,
			MasterKeys: masterKeys,
		})
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create service")
		}


		if len(masterKeys) > 1 {
			go func(){

				err := vault.RollDeks(context.Background())
				if err != nil{
					logger.Err(err).Msg("failed to roll deks")
				}
			}()
		}

		
		srv, err := connect.New(connect.Config{Logger: logger, Service: vault})
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create service")
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
