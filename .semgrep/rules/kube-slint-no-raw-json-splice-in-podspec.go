package fixtures

import (
	"encoding/json"
	"fmt"
)

func bad(name, ns, serviceAccountName string) string {
	// ruleid: kube-slint-no-raw-json-splice-in-podspec
	return fmt.Sprintf(`{"metadata":{"name":"%s","namespace":"%s"},"spec":{"serviceAccountName":"%s"}}`, name, ns, serviceAccountName)
}

type podMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type podSpec struct {
	ServiceAccountName string `json:"serviceAccountName"`
}

type podOverride struct {
	Metadata podMetadata `json:"metadata"`
	Spec     podSpec     `json:"spec"`
}

func good(name, ns, serviceAccountName string) (string, error) {
	// ok: kube-slint-no-raw-json-splice-in-podspec
	data, err := json.Marshal(podOverride{
		Metadata: podMetadata{Name: name, Namespace: ns},
		Spec:     podSpec{ServiceAccountName: serviceAccountName},
	})
	return string(data), err
}

func goodUnrelatedSprintf(name string) string {
	// ok: kube-slint-no-raw-json-splice-in-podspec
	return fmt.Sprintf("pod-%s-%d", name, 1)
}
