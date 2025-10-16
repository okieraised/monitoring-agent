package monitoring

import (
	"context"
	"fmt"
	"net/http"

	"github.com/arl/statsviz"
	"github.com/okieraised/monitoring-agent/internal/config"
	"github.com/okieraised/monitoring-agent/internal/constants"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

func getMonitoringPort() int {
	port := viper.GetInt(config.AgentMonitoringPort)
	if port <= 0 {
		return constants.AgentDefaultMonitoringPort
	}
	return port
}

func NewMonitoringServer(ctx context.Context) error {
	log.Default().Info("Starting monitoring server")
	vizServer := http.NewServeMux()
	err := statsviz.Register(vizServer)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	defer close(errCh)

	go func() {
		err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", getMonitoringPort()), vizServer)
		if err != nil {
			errCh <- err
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Default().Info("Shutting down monitoring server")
			return nil
		case err = <-errCh:
			log.Default().Error(errors.Wrap(err, "failed to start monitoring server").Error())
			return err
		}
	}
}
