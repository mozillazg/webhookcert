
set -e
set -o errtrace

function test_demo_validating_webhook_1() {
  kubectl create ns test-delete-ns || true
  kubectl delete ns test-delete-ns 2>&1 |grep 'denied by demo validating webhook'
}

function test_demo_validating_webhook_2() {
  kubectl create sa test-delete-sa || true
  kubectl delete sa test-delete-sa 2>&1 |grep 'denied by demo validating webhook'
}

function test_demo_mutating_webhook_1() {
  kubectl delete configmap test-create-configmap || true
  kubectl create configmap test-create-configmap --save-config=true || true
  kubectl get configmap test-create-configmap -o yaml |grep 'patched-by-demo-mutating-webhook'
}

function test_demo_mutating_webhook_2() {
  kubectl delete secret test-create-secret || true
  kubectl create secret generic test-create-secret  --save-config=true || true
  kubectl get secret test-create-secret -o yaml |grep 'patched-by-demo-mutating-webhook'
}

function main() {
  test_demo_validating_webhook_1
  test_demo_validating_webhook_2

  test_demo_mutating_webhook_1
  test_demo_mutating_webhook_2
}

main
