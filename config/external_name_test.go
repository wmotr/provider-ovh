/*
Copyright 2022 Upbound Inc.
*/

package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tf "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	ovhtf "github.com/ovh/terraform-provider-ovh/v2/ovh"
)

func TestUserIdentifierFromProvider(t *testing.T) {
	cases := map[string]struct {
		externalName string
		params       map[string]any
		want         string
		wantErr      string
	}{
		"EmptyExternalNameReturnsEmpty": {
			externalName: "",
			params:       map[string]any{"service_name": "svc-1"},
			want:         "",
		},
		"HappyPath": {
			externalName: "user-123",
			params:       map[string]any{"service_name": "svc-1"},
			want:         "svc-1/user-123",
		},
		"MissingServiceName": {
			externalName: "user-123",
			params:       map[string]any{},
			wantErr:      "service_name",
		},
		"WrongTypeServiceName": {
			externalName: "user-123",
			params:       map[string]any{"service_name": 42},
			wantErr:      "service_name",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := userIdentifierFromProvider.GetIDFn(context.Background(), tc.externalName, tc.params, nil)
			assertGetID(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestServiceNameIdentifierFromProvider(t *testing.T) {
	cases := map[string]struct {
		externalName string
		params       map[string]any
		want         string
		wantErr      string
	}{
		"EmptyExternalNameReturnsEmpty": {
			externalName: "",
			params:       map[string]any{"service_name": "svc-1"},
			want:         "",
		},
		"HappyPath": {
			externalName: "stream-1",
			params:       map[string]any{"service_name": "svc-1"},
			want:         "svc-1/stream-1",
		},
		"MissingServiceName": {
			externalName: "stream-1",
			params:       map[string]any{},
			wantErr:      "service_name",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := serviceNameIdentifierFromProvider.GetIDFn(context.Background(), tc.externalName, tc.params, nil)
			assertGetID(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestDatabaseClusterIdentifierFromProvider(t *testing.T) {
	cases := map[string]struct {
		externalName string
		params       map[string]any
		want         string
		wantErr      string
	}{
		"EmptyExternalNameReturnsEmpty": {
			externalName: "",
			params:       map[string]any{"service_name": "svc-1", "cluster_id": "cluster-1"},
			want:         "",
		},
		"HappyPath": {
			externalName: "user-1",
			params:       map[string]any{"service_name": "svc-1", "cluster_id": "cluster-1"},
			want:         "svc-1/cluster-1/user-1",
		},
		"MissingClusterID": {
			externalName: "user-1",
			params:       map[string]any{"service_name": "svc-1"},
			wantErr:      "cluster_id",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := databaseClusterIdentifierFromProvider.GetIDFn(context.Background(), tc.externalName, tc.params, nil)
			assertGetID(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestDatabaseLogSubscriptionIdentifierFromProvider(t *testing.T) {
	cases := map[string]struct {
		externalName string
		params       map[string]any
		want         string
		wantErr      string
	}{
		"EmptyExternalNameReturnsEmpty": {
			externalName: "",
			params:       map[string]any{"service_name": "svc-1", "cluster_id": "cluster-1", "engine": "kafka"},
			want:         "",
		},
		"HappyPath": {
			externalName: "sub-1",
			params:       map[string]any{"service_name": "svc-1", "cluster_id": "cluster-1", "engine": "kafka"},
			want:         "svc-1/kafka/cluster-1/sub-1",
		},
		"MissingEngine": {
			externalName: "sub-1",
			params:       map[string]any{"service_name": "svc-1", "cluster_id": "cluster-1"},
			wantErr:      "engine",
		},
		"WrongTypeEngine": {
			externalName: "sub-1",
			params:       map[string]any{"service_name": "svc-1", "cluster_id": "cluster-1", "engine": 42},
			wantErr:      "engine",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := databaseLogSubscriptionIdentifierFromProvider.GetIDFn(context.Background(), tc.externalName, tc.params, nil)
			assertGetID(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestKubeSDKIdentifiersUseRawProviderIDs(t *testing.T) {
	cases := map[string]struct {
		externalName string
		params       map[string]any
	}{
		"ovh_cloud_project_kube": {
			externalName: "7d058f8c-dd00-4e0f-995f-33656aab9e96",
			params:       map[string]any{"service_name": "project-id"},
		},
		"ovh_cloud_project_kube_nodepool": {
			externalName: "92c1ddfb-8d9f-4e22-b70b-5c446829fe1e",
			params: map[string]any{
				"service_name": "project-id",
				"kube_id":      "7d058f8c-dd00-4e0f-995f-33656aab9e96",
			},
		},
	}

	for resourceName, tc := range cases {
		t.Run(resourceName, func(t *testing.T) {
			cfg := TerraformPluginSDKExternalNameConfigs[resourceName]
			got, err := cfg.GetIDFn(context.Background(), tc.externalName, tc.params, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.externalName {
				t.Fatalf("SDK read ID = %q, want raw provider UUID %q", got, tc.externalName)
			}
			if !cfg.DisableNameInitializer {
				t.Fatal("provider-assigned IDs must keep the name initializer disabled")
			}
		})
	}
}

func TestKubeSDKRawIDsReachExactReadEndpoints(t *testing.T) {
	const (
		serviceName = "project-id"
		clusterID   = "7d058f8c-dd00-4e0f-995f-33656aab9e96"
		poolID      = "92c1ddfb-8d9f-4e22-b70b-5c446829fe1e"
	)

	cases := map[string]struct {
		resourceName string
		externalName string
		attributes   map[string]string
		responses    map[string]string
	}{
		"Cluster": {
			resourceName: "ovh_cloud_project_kube",
			externalName: clusterID,
			attributes: map[string]string{
				"service_name": serviceName,
				"name":         "gh-runners",
				"region":       "GRA11",
			},
			responses: map[string]string{
				"GET /cloud/project/" + serviceName + "/kube/" + clusterID:                  `{"id":"` + clusterID + `","name":"gh-runners","region":"GRA11","status":"READY","version":"1.33.10-4","updatePolicy":"MINIMAL_DOWNTIME","plan":"free","nextUpgradeVersions":[],"customization":{},"ipAllocationPolicy":{}}`,
				"POST /cloud/project/" + serviceName + "/kube/" + clusterID + "/kubeconfig": `{"content":"apiVersion: v1\nkind: Config\nclusters:\n- name: cluster\n  cluster:\n    server: https://example.invalid\n    certificate-authority-data: dummy\ncontexts: []\nusers:\n- name: user\n  user:\n    client-certificate-data: dummy\n    client-key-data: dummy\n"}`,
			},
		},
		"NodePool": {
			resourceName: "ovh_cloud_project_kube_nodepool",
			externalName: poolID,
			attributes: map[string]string{
				"service_name": serviceName,
				"kube_id":      clusterID,
				"name":         "workers",
				"flavor_name":  "b3-16",
			},
			responses: map[string]string{
				"GET /cloud/project/" + serviceName + "/kube/" + clusterID + "/nodepool/" + poolID: `{"id":"` + poolID + `","name":"workers","flavor":"b3-16","status":"READY","autoscale":true,"desiredNodes":4,"currentNodes":4,"availableNodes":4,"minNodes":0,"maxNodes":4,"monthlyBilled":false,"availabilityZones":[],"autoscaling":{}}`,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			requested := make(map[string]bool, len(tc.responses))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				key := r.Method + " " + r.URL.Path
				if key == "GET /auth/details" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{}`))
					return
				}
				body, ok := tc.responses[key]
				if !ok {
					t.Errorf("unexpected OVH request %s", key)
					http.NotFound(w, r)
					return
				}
				requested[key] = true
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()

			provider := ovhtf.Provider()
			diagnostics := provider.Configure(context.Background(), tf.NewResourceConfigRaw(map[string]any{
				"endpoint":     server.URL,
				"access_token": "test-token",
			}))
			if diagnostics.HasError() {
				t.Fatalf("configure OVH provider: %v", diagnostics)
			}

			cfg := TerraformPluginSDKExternalNameConfigs[tc.resourceName]
			id, err := cfg.GetIDFn(context.Background(), tc.externalName, map[string]any{
				"service_name": serviceName,
				"kube_id":      clusterID,
			}, nil)
			if err != nil {
				t.Fatalf("resolve SDK ID: %v", err)
			}
			state := &tf.InstanceState{ID: id, Attributes: tc.attributes}
			observed, refreshDiagnostics := provider.ResourcesMap[tc.resourceName].RefreshWithoutUpgrade(context.Background(), state, provider.Meta())
			if refreshDiagnostics.HasError() {
				t.Fatalf("refresh %s: %v", tc.resourceName, refreshDiagnostics)
			}
			if observed == nil || observed.ID != tc.externalName {
				t.Fatalf("observed ID = %v, want %s", observed, tc.externalName)
			}
			for endpoint := range tc.responses {
				if !requested[endpoint] {
					t.Errorf("expected OVH request %s was not made", endpoint)
				}
			}
		})
	}
}

func TestSDKMapBindingsRestored(t *testing.T) {
	cases := map[string]string{
		"ovh_cloud_project_user":                     "svc-1/u",
		"ovh_cloud_project_database_postgresql_user": "svc-1/cluster-1/u",
		"ovh_cloud_project_database_clickhouse_user": "svc-1/cluster-1/u",
		// service_name/engine/cluster_id/id
		"ovh_cloud_project_database_log_subscription": "svc-1/postgresql/cluster-1/u",
		// service_name/kube_id/subscription_id
		"ovh_cloud_project_kube_log_subscription": "svc-1/kube-1/u",
		// service_name/stream_id
		"ovh_dbaas_logs_output_graylog_stream": "svc-1/u",
	}

	params := map[string]any{
		"service_name": "svc-1",
		"cluster_id":   "cluster-1",
		"user_id":      "uid",
		"region":       "GRA9",
		"engine":       "postgresql",
		"kube_id":      "kube-1",
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			cfg, ok := TerraformPluginSDKExternalNameConfigs[name]
			if !ok {
				t.Fatalf("%s missing from TerraformPluginSDKExternalNameConfigs", name)
			}
			got, err := cfg.GetIDFn(context.Background(), "u", params, nil)
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", name, err)
			}
			if got != want {
				t.Errorf("%s: got %q, want %q (regression — entry likely set to IdentifierFromProvider)", name, got, want)
			}
		})
	}
}

func TestPostgresqlUserIdentifierFromProvider(t *testing.T) {
	t.Run("UsesExternalNameWhenPresent", func(t *testing.T) {
		got, err := postgresqlUserIdentifierFromProvider.GetIDFn(context.Background(), "user-123", map[string]any{
			"service_name": "svc-1",
			"cluster_id":   "cluster-1",
		}, nil)
		assertGetID(t, got, err, "svc-1/cluster-1/user-123", "")
	})

	t.Run("ResolvesMissingExternalNameFromOVH", func(t *testing.T) {
		server := newPostgresqlUserLookupServer(t, map[string]string{
			"user-111": "someone-else",
			"user-123": "johndoe",
		})
		defer server.Close()

		// Use nested "configuration" structure matching production ts.Map() format
		got, err := postgresqlUserIdentifierFromProvider.GetIDFn(context.Background(), "", map[string]any{
			"service_name": "svc-1",
			"cluster_id":   "cluster-1",
			"name":         "johndoe",
		}, map[string]any{
			"configuration": map[string]any{
				"endpoint":     server.URL,
				"access_token": "token",
			},
		})
		assertGetID(t, got, err, "svc-1/cluster-1/user-123", "")
	})

	t.Run("ResolvesMissingExternalNameFromOVH_FlatProviderConfig", func(t *testing.T) {
		// Also works when providerConfig is flat (e.g. direct unit-test use)
		server := newPostgresqlUserLookupServer(t, map[string]string{
			"user-111": "someone-else",
			"user-123": "johndoe",
		})
		defer server.Close()

		got, err := postgresqlUserIdentifierFromProvider.GetIDFn(context.Background(), "", map[string]any{
			"service_name": "svc-1",
			"cluster_id":   "cluster-1",
			"name":         "johndoe",
		}, map[string]any{
			"endpoint":     server.URL,
			"access_token": "token",
		})
		assertGetID(t, got, err, "svc-1/cluster-1/user-123", "")
	})

	t.Run("LookupMissReturnsEmpty", func(t *testing.T) {
		server := newPostgresqlUserLookupServer(t, map[string]string{
			"user-111": "someone-else",
		})
		defer server.Close()

		got, err := postgresqlUserIdentifierFromProvider.GetIDFn(context.Background(), "", map[string]any{
			"service_name": "svc-1",
			"cluster_id":   "cluster-1",
			"name":         "johndoe",
		}, map[string]any{
			"configuration": map[string]any{
				"endpoint":     server.URL,
				"access_token": "token",
			},
		})
		assertGetID(t, got, err, "", "")
	})

	t.Run("MissingClusterIDWithoutExternalNameReturnsEmpty", func(t *testing.T) {
		got, err := postgresqlUserIdentifierFromProvider.GetIDFn(context.Background(), "", map[string]any{
			"service_name": "svc-1",
			"name":         "johndoe",
		}, nil)
		assertGetID(t, got, err, "", "")
	})

	t.Run("MissingClusterID", func(t *testing.T) {
		got, err := postgresqlUserIdentifierFromProvider.GetIDFn(context.Background(), "user-123", map[string]any{
			"service_name": "svc-1",
		}, nil)
		assertGetID(t, got, err, "", "cluster_id")
	})
}

func newPostgresqlUserLookupServer(t *testing.T, users map[string]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		path := strings.TrimPrefix(r.URL.Path, "/1.0")
		if path == "/cloud/project/svc-1/database/postgresql/cluster-1/user" {
			ids := make([]string, 0, len(users))
			for id := range users {
				ids = append(ids, id)
			}
			if err := json.NewEncoder(w).Encode(ids); err != nil {
				t.Fatalf("encode ids: %v", err)
			}
			return
		}

		prefix := "/cloud/project/svc-1/database/postgresql/cluster-1/user/"
		if strings.HasPrefix(path, prefix) {
			id := strings.TrimPrefix(path, prefix)
			username, ok := users[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]string{"id": id, "username": username}); err != nil {
				t.Fatalf("encode user: %v", err)
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func assertGetID(t *testing.T, got string, err error, want, wantErr string) {
	t.Helper()
	if wantErr != "" {
		if err == nil {
			t.Fatalf("expected error containing %q, got nil (id=%q)", wantErr, got)
		}
		if !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("error %q does not contain %q", err.Error(), wantErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
