package report_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/report"
	"github.com/stretchr/testify/assert"
)

func TestLocation_FullyQualifiedName(t *testing.T) {
	cases := []struct {
		name string
		loc  report.Location
		want string
	}{
		{
			name: "workload without container",
			loc:  report.Location{Kind: "Deployment", Namespace: "hello-system", Name: "hello-operator"},
			want: "Deployment/hello-system/hello-operator",
		},
		{
			name: "container scoped",
			loc: report.Location{
				Kind: "Deployment", Namespace: "hello-system", Name: "hello-operator", Container: "hello-operator",
			},
			want: "Deployment/hello-system/hello-operator/container/hello-operator",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.loc.FullyQualifiedName())
		})
	}
}
