package engine

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "kubevault.dev/operator/apis/engine/v1alpha1"
)

const gcpPolicyTest1 = `
path "gcp/config" {
	capabilities = ["create", "update", "read", "delete"]
}

path "gcp/roleset/*" {
	capabilities = ["create", "update", "read", "delete"]
}

path "gcp/token/*" {
	capabilities = ["create", "update", "read"]
}

path "gcp/key/*" {
	capabilities = ["create", "update", "read"]
}
`
const gcpPolicyTest2 = `
path "my-gcp-path/config" {
	capabilities = ["create", "update", "read", "delete"]
}

path "my-gcp-path/roleset/*" {
	capabilities = ["create", "update", "read", "delete"]
}

path "my-gcp-path/token/*" {
	capabilities = ["create", "update", "read"]
}

path "my-gcp-path/key/*" {
	capabilities = ["create", "update", "read"]
}
`

func NewFakeVaultPolicyServer() *httptest.Server {
	router := mux.NewRouter()

	router.HandleFunc("/v1/sys/policies/acl/{path}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		path := vars["path"]
		body := r.Body
		data, _ := ioutil.ReadAll(body)
		var newdata map[string]interface{}
		_ = json.Unmarshal(data, &newdata)

		if path == "k8s.-.demo.gcpse" {
			if newdata["policy"] == gcpPolicyTest1 || newdata["policy"] == gcpPolicyTest2 {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}).Methods(http.MethodPut)

	return httptest.NewServer(router)
}

func TestSecretEngine_CreatePolicy(t *testing.T) {

	srv := NewFakeVaultPolicyServer()
	defer srv.Close()

	tests := []struct {
		name         string
		secretEngine *api.SecretEngine
		path         string
		wantErr      bool
	}{
		{
			name: "Create policy for gcp secret engine",
			path: "gcp",
			secretEngine: &api.SecretEngine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gcpse",
					Namespace: "demo",
				},
				Spec: api.SecretEngineSpec{
					VaultRef: v1.LocalObjectReference{},
					Path:     "",
					SecretEngineConfiguration: api.SecretEngineConfiguration{
						GCP: &api.GCPConfiguration{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Create policy for my-gcp-path secret engine",
			path: "my-gcp-path",
			secretEngine: &api.SecretEngine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gcpse",
					Namespace: "demo",
				},
				Spec: api.SecretEngineSpec{
					VaultRef: v1.LocalObjectReference{},
					Path:     "",
					SecretEngineConfiguration: api.SecretEngineConfiguration{
						GCP: &api.GCPConfiguration{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Create policy for gcp secret engine failed",
			path: "my-gcp-path",
			secretEngine: &api.SecretEngine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gcpse",
					Namespace: "demo",
				},
				Spec: api.SecretEngineSpec{
					VaultRef: v1.LocalObjectReference{},
					Path:     "",
					SecretEngineConfiguration: api.SecretEngineConfiguration{
						AWS: &api.AWSConfiguration{},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			vc, err := vaultClient(srv.URL, "root")
			assert.Nil(t, err, "failed to create vault client")

			seClient := &SecretEngine{
				secretEngine: tt.secretEngine,
				path:         tt.path,
				vaultClient:  vc,
			}
			if err := seClient.CreatePolicy(); (err != nil) != tt.wantErr {
				t.Errorf("CreatePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}