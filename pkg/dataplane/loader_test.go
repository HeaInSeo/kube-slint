package dataplane_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func TestLoadDir_MultiDocumentSingleFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.yaml", `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: ns
spec:
  selector:
    matchLabels: {app: app}
  template:
    metadata:
      labels: {app: app}
    spec:
      containers:
        - name: app
          ports:
            - name: metrics
              containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: app-metrics
  namespace: ns
spec:
  selector: {app: app}
  ports:
    - name: metrics
      port: 8080
      targetPort: 8080
`)

	b, warnings, err := dataplane.LoadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, b.Workloads, 1)
	require.Len(t, b.Services, 1)
	assert.Equal(t, "app", b.Workloads[0].Metadata.Name)
	assert.Equal(t, "app-metrics", b.Services[0].Metadata.Name)
	assert.Equal(t, "app.yaml", b.Workloads[0].SourceFile)
}

func TestLoadDir_UnknownKindIsNotError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "ns.yaml", `
apiVersion: v1
kind: Namespace
metadata:
  name: hello-system
`)

	b, warnings, err := dataplane.LoadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, b.Unknown, 1)
	assert.Equal(t, "Namespace", b.Unknown[0].Kind)
	assert.Equal(t, "hello-system", b.Unknown[0].Name)
}

func TestLoadDir_MalformedDocumentIsWarningNotFatal(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.yaml", "kind: [unterminated\n")
	writeFile(t, dir, "good.yaml", `
apiVersion: v1
kind: Service
metadata:
  name: svc
  namespace: ns
spec: {}
`)

	b, warnings, err := dataplane.LoadDir(dir)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Equal(t, "bad.yaml", warnings[0].File)
	require.Len(t, b.Services, 1)
	assert.Equal(t, "svc", b.Services[0].Metadata.Name)
}

func TestLoadDir_TrailingEmptyDocumentSkipped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "svc.yaml", `
apiVersion: v1
kind: Service
metadata:
  name: svc
  namespace: ns
spec: {}
---
`)

	b, warnings, err := dataplane.LoadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Len(t, b.Services, 1)
}

func TestLoadDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	b, warnings, err := dataplane.LoadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Empty(t, b.Workloads)
	assert.Empty(t, b.Services)
	assert.Empty(t, b.ServiceMonitors)
	assert.Empty(t, b.Unknown)
}

func TestLoadDir_NonexistentDir(t *testing.T) {
	_, _, err := dataplane.LoadDir(filepath.Join(t.TempDir(), "does-not-exist"))
	assert.Error(t, err)
}

func TestLoadDir_StatefulSetAndDaemonSetShareWorkloadShape(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sts.yaml", `
apiVersion: apps/v1
kind: StatefulSet
metadata: {name: sts, namespace: ns}
spec:
  template:
    spec:
      containers: [{name: c}]
`)
	writeFile(t, dir, "ds.yaml", `
apiVersion: apps/v1
kind: DaemonSet
metadata: {name: ds, namespace: ns}
spec:
  template:
    spec:
      containers: [{name: c}]
`)

	b, warnings, err := dataplane.LoadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Len(t, b.Workloads, 2)
}

func TestLoadDir_ServiceMonitor(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sm.yaml", `
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata: {name: sm, namespace: ns}
spec:
  selector:
    matchLabels: {app: app}
  endpoints:
    - port: metrics
`)

	b, warnings, err := dataplane.LoadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, b.ServiceMonitors, 1)
	assert.Equal(t, "app", b.ServiceMonitors[0].Spec.Selector.MatchLabels["app"])
	require.Len(t, b.ServiceMonitors[0].Spec.Endpoints, 1)
	assert.Equal(t, "metrics", b.ServiceMonitors[0].Spec.Endpoints[0].Port)
}
