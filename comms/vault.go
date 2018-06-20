package comms

import (
	"errors"
	"fmt"
	"strings"

	vault "github.com/hashicorp/vault/api"
)

var (
	ErrInvalidVaultMount = errors.New("vault mount is invalid")
	ErrInvalidSecretType = errors.New("vault secret is invalid type")
)

// VKVault is a wrapper around Vault
type VKVault struct {
	Client *vault.Client
}

// VKVaultMounts is a slice of VKVaultMount's
type VKVaultMounts []VKVaultMount

// VKVaultMount is a representation of a mount we care about
type VKVaultMount struct {
	MountPath    string
	KeysPath     string
	MountPointer *vault.MountOutput
	Secrets      *VKVaultSecrets
}

// VKVaultSecrets is a slice of VKVaultSecret's
type VKVaultSecrets []VKVaultSecret

// VKVaultSecret is a kv secret stored in vault
type VKVaultSecret struct {
	Name       string
	Namespace  string
	SecretType string
	Pairs      map[string]string
}

// NewVKVaultClient returns a new VKVault client
func NewVKVaultClient() (*VKVault, error) {
	config := vault.DefaultConfig()
	client, err := vault.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &VKVault{
		Client: client,
	}, nil
}

// GetMounts will return a list of VKVaultMounts based on mountPath
func (v *VKVault) GetMounts(mountPath string) (*VKVaultMounts, error) {
	mounts := new(VKVaultMounts)
	mountMap, err := v.Client.Sys().ListMounts()
	if err != nil {
		return nil, err
	}

	for mount, pointer := range mountMap {
		if strings.HasPrefix(strings.Trim(mountPath, "/"), mount) && pointer.Type == "kv" {
			vaultMount := &VKVaultMount{
				MountPath:    mount,
				KeysPath:     mountPath,
				MountPointer: pointer,
			}
			vaultMount.populateSecrets(v)
			*mounts = append(*mounts, *vaultMount)
		}
	}

	return mounts, nil
}

// populateSecrets will return VKVaultSecrets when given a VKVaultMount
func (m *VKVaultMount) populateSecrets(v *VKVault) (*VKVaultMount, error) {
	returnSecrets := new(VKVaultSecrets)
	namespaces, err := v.Client.Logical().List(m.KeysPath)
	if err != nil {
		return nil, err
	}
	if namespaces == nil {
		return nil, nil
	}
	// List namespaces loop
	for _, data := range namespaces.Data["keys"].([]interface{}) {
		namespace := strings.Trim(data.(string), "/")

		secretTypes, err := v.Client.Logical().List(fmt.Sprintf("%s/%s", m.KeysPath, namespace))
		if err != nil {
			return nil, err
		}
		// List secretsTypes loop
		for _, data := range secretTypes.Data["keys"].([]interface{}) {
			secretType := strings.Trim(data.(string), "/")

			if !(secretType == "secrets" ||
				secretType == "configmaps") {
				continue
			}

			secretsTypesPath := fmt.Sprintf("%s/%s/%s", m.KeysPath, namespace, secretType)
			secretsList, err := v.Client.Logical().List(secretsTypesPath)
			if err != nil {
				return nil, err
			}
			// List secrets loop
			for _, data := range secretsList.Data["keys"].([]interface{}) {
				secretName := strings.Trim(data.(string), "/")
				secretPath := fmt.Sprintf("%s/%s", secretsTypesPath, secretName)

				secretMap, err := v.Client.Logical().Read(secretPath)
				if err != nil {
					return nil, err
				}
				appendSecret := &VKVaultSecret{
					Name:       secretName,
					Namespace:  namespace,
					SecretType: secretType,
					Pairs:      make(map[string]string),
				}

				if secretMap != nil {
					for key, value := range secretMap.Data {
						appendSecret.Pairs[key] = value.(string)
					}
				}
				*returnSecrets = append(*returnSecrets, *appendSecret)
			}
		}
	}
	m.Secrets = returnSecrets
	return m, nil
}
