package pipelines

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/redhat-developer/kam/pkg/pipelines/config"
	"github.com/redhat-developer/kam/pkg/pipelines/ioutils"
	"github.com/redhat-developer/kam/test"
)

func TestAddEnv(t *testing.T) {
	fakeFs := ioutils.NewMemoryFilesystem()
	gitopsPath := afero.GetTempDir(fakeFs, "test")
	pipelinesFile := filepath.Join(gitopsPath, pipelinesFile)
	envParameters := EnvParameters{
		PipelinesFolderPath: gitopsPath,
		EnvName:             "dev",
	}
	_ = afero.WriteFile(fakeFs, pipelinesFile, []byte("environments:"), 0644)

	if err := AddEnv(&envParameters, fakeFs); err != nil {
		t.Fatalf("AddEnv() failed :%s", err)
	}

	wantedPaths := []string{
		"environments/dev/env/base/kustomization.yaml",
		"environments/dev/env/base/dev-environment.yaml",
		"environments/dev/env/overlays/kustomization.yaml",
	}
	for _, path := range wantedPaths {
		t.Run(fmt.Sprintf("checking path %s already exists", path), func(rt *testing.T) {
			// The inmemory version of Afero doesn't return errors
			exists, _ := fakeFs.Exists(filepath.Join(gitopsPath, path))
			if !exists {
				t.Fatalf("The file is not present at path : %v", path)
			}
		})
	}

	got := mustReadFileAsMap(t, fakeFs, pipelinesFile)
	want := map[string]interface{}{
		"environments": []interface{}{
			map[string]interface{}{
				"name": "dev",
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("written environments failed:\n%s", diff)
	}
}

func TestAddEnvWithClusterProvided(t *testing.T) {
	fakeFs := ioutils.NewMemoryFilesystem()
	gitopsPath := afero.GetTempDir(fakeFs, "test")
	pipelinesFilePath := filepath.Join(gitopsPath, pipelinesFile)
	envParameters := EnvParameters{
		PipelinesFolderPath: gitopsPath,
		EnvName:             "dev",
		Cluster:             "testing.cluster",
	}
	_ = afero.WriteFile(fakeFs, pipelinesFilePath, []byte("environments:"), 0644)

	if err := AddEnv(&envParameters, fakeFs); err != nil {
		t.Fatalf("AddEnv() failed :%s", err)
	}

	wantedPaths := []string{
		"environments/dev/env/base/kustomization.yaml",
		"environments/dev/env/base/dev-environment.yaml",
		"environments/dev/env/overlays/kustomization.yaml",
	}
	for _, path := range wantedPaths {
		t.Run(fmt.Sprintf("checking path %s already exists", path), func(rt *testing.T) {
			// The inmemory version of Afero doesn't return errors
			exists, _ := fakeFs.Exists(filepath.Join(gitopsPath, path))
			if !exists {
				t.Fatalf("The file is not present at path : %v", path)
			}

		})
	}

	got := mustReadFileAsMap(t, fakeFs, pipelinesFilePath)
	want := map[string]interface{}{
		"environments": []interface{}{
			map[string]interface{}{
				"cluster": "testing.cluster",
				"name":    "dev",
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("written environments failed:\n%s", diff)
	}
}

func TestAddEnvWithExistingName(t *testing.T) {
	fakeFs := ioutils.NewMemoryFilesystem()
	gitopsPath := afero.GetTempDir(fakeFs, "test")

	pipelinesFile := filepath.Join(gitopsPath, pipelinesFile)
	envParameters := EnvParameters{
		PipelinesFolderPath: gitopsPath,
		EnvName:             "dev",
	}
	_ = afero.WriteFile(fakeFs, pipelinesFile, []byte("environments:\n - name: dev\n"), 0644)

	if err := AddEnv(&envParameters, fakeFs); err == nil {
		t.Fatal("AddEnv() did not fail with duplicate environment")
	}
}

func TestNewEnvironment(t *testing.T) {
	tests := []struct {
		m      *config.Manifest
		name   string
		errMsg string
		want   *config.Environment
	}{
		{
			m: &config.Manifest{
				GitOpsURL: "https://github.com/foo/bar",
				Config: &config.Config{
					Pipelines: &config.PipelinesConfig{
						Name: "my-cicd",
					},
				},
				Environments: []*config.Environment{
					{
						Name: "myenv1",
					},
				},
			},
			name: "test-env",
			want: &config.Environment{
				Name: "test-env",
				Pipelines: &config.Pipelines{
					Integration: &config.TemplateBinding{
						Template: appCITemplateName,
						Bindings: []string{"github-push-binding"},
					},
				},
			},
		},
		{
			m: &config.Manifest{
				GitOpsURL: "https://gitlab.com/foo/bar",
				Config: &config.Config{
					Pipelines: &config.PipelinesConfig{
						Name: "my-cicd",
					},
				},
				Environments: []*config.Environment{
					{
						Name: "my-cicd",
					},
				},
			},
			name: "test-env",
			want: &config.Environment{
				Name: "test-env",
				Pipelines: &config.Pipelines{
					Integration: &config.TemplateBinding{
						Template: appCITemplateName,
						Bindings: []string{"gitlab-push-binding"},
					},
				},
			},
		},
		{
			m: &config.Manifest{
				// no GitOpsURL -> no Pipelines
				Config: &config.Config{
					Pipelines: &config.PipelinesConfig{
						Name: "my-cicd",
					},
				},
				Environments: []*config.Environment{
					{
						Name: "my-env2",
					},
				},
			},
			name: "test-env",
			want: &config.Environment{
				Name: "test-env",
			},
		},
		{
			m: &config.Manifest{
				GitOpsURL: "https://gitlab.com/foo/bar",
				Environments: []*config.Environment{
					{
						// no CICD -> no Pipelines
						Name: "my-env4",
					},
				},
			},
			name: "test-env",
			want: &config.Environment{
				Name: "test-env",
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test_%d", i), func(rt *testing.T) {
			got, err := newEnvironment(tt.m, tt.name)

			if !test.ErrorMatch(rt, tt.errMsg, err) {
				rt.Errorf("err mismatch want: %s got: %s: \n", tt.errMsg, err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				rt.Errorf("env mismatch: \n%s", diff)
			}
		})
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func mustReadFileAsMap(t *testing.T, fs afero.Fs, filename string) map[string]interface{} {
	t.Helper()
	b, err := afero.ReadFile(fs, filename)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]interface{}{}
	err = yaml.Unmarshal(b, &m)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
