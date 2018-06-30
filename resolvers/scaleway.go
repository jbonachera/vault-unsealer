package resolvers

import (
	"fmt"

	scaleway "github.com/scaleway/go-scaleway"
	"github.com/scaleway/go-scaleway/logger"
	config "github.com/scaleway/scaleway-cli/pkg/config"
)

type Resolver interface {
	Resolve(privateIP string) (publicIP string)
}

type Scaleway struct {
	api *scaleway.ScalewayAPI
}

func NewScalewayResolver(region string) (Resolver, error) {
	config, cfgErr := config.GetConfig()
	if cfgErr != nil {
		return nil, fmt.Errorf("unable to open .scwrc config file: %v", cfgErr)
	}

	api, err := scaleway.NewScalewayAPI(config.Organization, config.Token, "go-http-client", region)
	if err != nil {
		return nil, err
	}
	api.Logger = logger.NewDisableLogger()
	return &Scaleway{
		api: api,
	}, nil
}

func (s *Scaleway) Resolve(privateIP string) string {
	set, err := s.api.GetServers(false, 0)
	if err != nil {
		return ""
	}
	for _, server := range *set {
		if server.PrivateIP == privateIP {
			return server.PublicAddress.IP
		}
	}
	return ""
}
