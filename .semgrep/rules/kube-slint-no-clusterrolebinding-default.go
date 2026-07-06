package fixtures

const badRBACTemplate = `apiVersion: rbac.authorization.k8s.io/v1
// ruleid: kube-slint-no-clusterrolebinding-default
kind: ClusterRoleBinding
metadata:
  name: kube-slint-scraper
`

const goodRBACTemplate = `apiVersion: rbac.authorization.k8s.io/v1
// ok: kube-slint-no-clusterrolebinding-default
kind: RoleBinding
metadata:
  name: kube-slint-scraper
  namespace: my-namespace
`
