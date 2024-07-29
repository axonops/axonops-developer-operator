# axonops-developer-operator

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.1.0](https://img.shields.io/badge/AppVersion-v0.1.0-informational?style=flat-square)

This helm chart installs the AxonOps Developer Operator to install a local Apache Cassandra cluster that also uses AxonOps for monitoring.
## About AxonOps https://axonops.com

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| AxonOps Limited | <info@axonops.com> |  |

## Requirements

Kubernetes: `>= 1.24.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| args | list | `[]` |  |
| enableDbSecrets | bool | `true` |  |
| env | list | `[]` |  |
| environmentSecret | string | `""` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"ghcr.io/axonops.com/axonops-developer-operator"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| manageCrds | bool | `true` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podMonitor.enabled | bool | `false` |  |
| podMonitor.labels | object | `{}` |  |
| podSecurityContext | object | `{}` |  |
| prometheusRules.additionalRuleAnnotations | object | `{}` |  |
| prometheusRules.additionalRuleLabels | object | `{}` |  |
| prometheusRules.enabled | bool | `false` |  |
| replicaCount | int | `1` |  |
| resources | object | `{}` |  |
| secretEnv | list | `[]` |  |
| securityContext | object | `{}` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `""` |  |
| tolerations | list | `[]` |  |
| volumeMounts | list | `[]` |  |
| volumes | list | `[]` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)