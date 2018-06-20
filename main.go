package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dix-icomys/vaultingkube/comms"
)

func main() {
	rootMountPath := os.Getenv("VK_VAULT_ROOT_MOUNT_PATH")
	if rootMountPath == "" {
		log.Fatal("Must set VK_VAULT_ROOT_MOUNT_PATH")
	}

	syncPeriod := os.Getenv("VK_SYNC_PERIOD")
	if syncPeriod == "" {
		syncPeriod = "300"
	}

	syncPeriodInt, err := strconv.Atoi(syncPeriod)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Started with sync period every %s seconds\n", syncPeriod)
	run()
	for range time.Tick(time.Second * time.Duration(syncPeriodInt)) {
		run()
	}
}

func run() {
	logger := logrus.New()

	vkVault, err := comms.NewVKVaultClient()
	if err != nil {
		logger.Fatal(err)
	}

	vkKube, err := comms.NewVKKubeClient(logger)
	if err != nil {
		logger.Fatal(err)
	}

	mounts, err := vkVault.GetMounts(os.Getenv("VK_VAULT_ROOT_MOUNT_PATH"))
	if err != nil {
		logger.Fatal(err)
	} else {
		for _, vaultSecret := range *mounts {
			if vaultSecret.Secrets != nil {
				for _, secret := range *vaultSecret.Secrets {
					logger.Infof("Found Secret %s", secret.Name)

					if vkKube.IsManaged(secret.Name, secret.SecretType, secret.Namespace) {
						if secret.SecretType == "secrets" {
							logger.Infof("Found Secret %s is secrets", secret.Name)
							err := vkKube.SetSecret(secret.Name, secret.Namespace, secret.Pairs)
							if err != nil {
								logger.Error(err)
							} else {
								logger.Infof("Set Secret for %s/%s", secret.Namespace, secret.Name)
							}
						} else if secret.SecretType == "configmaps" {
							logger.Infof("Found Secret %s is configmaps", secret.Name)
							err := vkKube.SetCM(secret.Name, secret.Namespace, secret.Pairs)
							if err != nil {
								logger.Error(err)
							} else {
								logger.Infof("Set ConfigMap for %s/%s", secret.Namespace, secret.Name)
							}
						}
					} else {
						if secret.SecretType == "secrets" {
							logger.Infof("Secret %s in namespace %s is not managed by VaultingKube, ignoring", secret.Name, secret.Namespace)
						} else if secret.SecretType == "configmaps" {
							logger.Infof("ConfigMap %s in namespace %s is not managed by VaultingKube, ignoring", secret.Name, secret.Namespace)
						}
					}
				}
			}
		}

		deleteOld := os.Getenv("VK_DELETE_OLD")
		if deleteOld == "" || deleteOld == "true" {
			err := vkKube.DeleteOld(mounts)
			if err != nil {
				logger.Fatal(err)
			}
		}
	}
}
