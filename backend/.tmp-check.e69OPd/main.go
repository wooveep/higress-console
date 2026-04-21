package main

import (
  "context"
  "encoding/json"
  "fmt"

  gatewaysvc "github.com/wooveep/aigateway-console/backend/internal/service/gateway"
  k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
)

func main() {
  svc := gatewaysvc.New(k8sclient.New(k8sclient.Config{Enabled:true, Namespace:"aigateway-system", KubectlBin:"kubectl", ResourcePrefix:"aigw-console", IngressClass:"aigateway"}))
  for _, name := range []string{"qwen", "doubao"} {
    target := fmt.Sprintf("ai-route-%s.internal", name)
    aliases := []string{
      fmt.Sprintf("ai-route-%s.internal-internal", name),
      fmt.Sprintf("ai-route-%s-fallback.internal", name),
      fmt.Sprintf("ai-route-%s-fallback.internal-internal", name),
    }
    items, err := svc.ListPluginInstances(context.Background(), "route", target, aliases...)
    if err != nil { panic(err) }
    out, _ := json.MarshalIndent(items, "", "  ")
    fmt.Printf("== %s ==\n%s\n", name, out)
  }
}
