package main

import (
	"fmt"
	"log"
	"strings"
	"syscall"

	consul "github.com/hashicorp/consul/api"
	vault "github.com/hashicorp/vault/api"
	"github.com/jbonachera/vault-unsealer/resolvers"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

func isVaultSealed(service *consul.ServiceEntry) bool {
	for _, check := range service.Checks {
		if check.Name == "Vault Sealed Status" {
			return check.Status == consul.HealthCritical
		}
	}
	return false
}

func discoverVaultAddr(client *consul.Client) []string {
	opt := &consul.QueryOptions{}
	services, _, err := client.Health().Service("vault", "standby", false, opt)
	if err != nil {
		panic(err)
	}
	out := make([]string, 0, len(services))
	for _, service := range services {
		if isVaultSealed(service) {
			out = append(out, service.Service.Address)
		}
	}
	return out
}

type Resolver interface {
	Resolve(privateIP string) (publicIP string)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "vault-unsealer",
		Short: "Discover sealed vault members using Consul, and unseal them all using the provided unseal-key.",
		Run: func(cmd *cobra.Command, _ []string) {
			var resolver Resolver
			userResolver, err := cmd.Flags().GetString("resolver")
			if err == nil {
				switch userResolver {
				case "scaleway":
					resolver, err = resolvers.NewScalewayResolver("par1")
					if err != nil {
						log.Printf("failed to start scaleway resolver: %v", err)
						return
					}
				}
			}
			consulConfig := consul.DefaultConfig()
			consulAPI, err := consul.NewClient(consulConfig)
			if err != nil {
				panic(err)
			}
			addresses := discoverVaultAddr(consulAPI)
			if len(addresses) == 0 {
				log.Println("no sealed vault servers were discovered")
				return
			}
			log.Printf("discovered vault servers: %s", strings.Join(addresses, ", "))
			fmt.Printf("Please enter the unseal key: ")
			bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Fatalf("failed to read unseal key from stdin: %v", err)
			}
			fmt.Println("")
			unsealKey := string(bytePassword)

			for _, address := range addresses {
				var target string
				if resolver != nil {
					target = resolver.Resolve(address)
				} else {
					target = address
				}
				target = fmt.Sprintf("http://%s:%d", target, 8200)
				if target == "" {
					log.Printf("WARN: failed to resolve %s address", address)
					continue
				}
				config := vault.DefaultConfig()
				config.Address = target
				log.Printf("INFO: Attempting to unseal Vault server at %s", target)
				api, err := vault.NewClient(config)
				if err != nil {
					log.Printf("skipping server %s: %v", address, err)
					continue
				}
				resp, err := api.Sys().Unseal(unsealKey)
				if err != nil {
					log.Printf("unsealing %s failed: %v", address, err)
					continue
				}
				log.Printf("%s: %d/%d", address, resp.N, resp.T)
			}
		},
	}
	rootCmd.Flags().StringP("resolver", "r", "", "resolve Vault adresses using the given resolver")
	rootCmd.Execute()
}
