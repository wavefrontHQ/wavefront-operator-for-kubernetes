pipeline {
  agent any

  tools {
    dockerTool 'Docker 20.10.20'
  }


  environment {
    GITHUB_CREDS_PSW = credentials("GITHUB_TOKEN")
    HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability-robot")
    PREFIX = 'projects.registry.vmware.com/tanzu_observability'
    DOCKER_IMAGE = "kubernetes-operator-snapshot"
    VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
    WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
  }

  parameters {
      string(name: 'OPERATOR_YAML_RC_SHA', defaultValue: '')
  }

  stages {
//     stage("Test Go Code") {
//       tools {
//         go 'Go 1.17'
//       }
//       steps {
//         withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
//           sh 'make checkfmt vet test'
//         }
//       }
//     }
    stage("Setup For Publish") {
      tools {
        go 'Go 1.17'
      }
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
        GKE_CLUSTER_NAME = "k8po-jenkins-ci"
      }
      steps {
        withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          sh './hack/jenkins/setup-for-integration-test.sh'
          sh './hack/jenkins/install_docker_buildx.sh'
          sh 'make semver-cli'
        }
      }
    }
    stage("Publish") {
      tools {
        go 'Go 1.17'
      }
      environment {
        RELEASE_TYPE = "alpha"
      }
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
          sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
          sh 'make docker-xplatform-build'
        }
      }
    }
    stage("Update RC branch") {
      tools {
        go 'Go 1.17'
      }
      environment {
        RELEASE_TYPE = "alpha"
        TOKEN = credentials('GITHUB_TOKEN')
      }
      steps {
        withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          sh './hack/jenkins/create-rc-ci.sh'
          script {
            env.OPERATOR_YAML_RC_SHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
          }
        }
      }
    }

    stage("Run Integration Tests") {
      environment {
        DEPLOY_SOURCE="rc"
      }

      parallel {
        stage("GKE Integration Test") {
          agent {
            label "gke"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          tools {
            go 'Go 1.17'
            dockerTool 'Docker 20.10.20'
          }
          environment {
            GKE_CLUSTER_NAME = "k8po-jenkins-ci"
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
          }
          stages {
            stage("run E2E") {
              steps {
                withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
                  sh './hack/jenkins/setup-for-integration-test.sh'
                  sh './hack/jenkins/install_docker_buildx.sh'
                  sh 'make semver-cli'
                  lock("integration-test-gke") {
                      sh 'make gke-connect-to-cluster'
                      sh 'make clean-cluster'
//                       sh 'make integration-test'
//                       sh 'make clean-cluster'
                  }
                }
              }
            }

            stage("run E2E with customization") {
              environment {
                KUSTOMIZATION_SOURCE="custom"
                NS="custom-namespace"
                SOURCE_PREFIX="projects.registry.vmware.com/tanzu_observability"
                PREFIX="projects.registry.vmware.com/tanzu_observability_keights_saas"
                HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
              }
              steps {
                lock("integration-test-gke") {
                  sh 'docker logout $PREFIX'
                  sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
                  sh 'make docker-copy-images'
                  sh 'make clean-cluster'
                  sh 'make integration-test'
                  sh 'make clean-cluster'
                }
              }
            }
          }
        }

//         stage("EKS Integration Test") {
//           agent {
//             label "eks"
//           }
//           options {
//             timeout(time: 30, unit: 'MINUTES')
//           }
//           tools {
//             go 'Go 1.17'
//           }
//           environment {
//             GCP_CREDS = credentials("GCP_CREDS")
//             AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
//             AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
//           }
//           steps {
//             withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
//               sh './hack/jenkins/setup-for-integration-test.sh'
//               sh './hack/jenkins/install_docker_buildx.sh'
//               sh 'make semver-cli'
//               lock("integration-test-eks") {
//                   sh 'make target-eks'
//                   sh 'make clean-cluster'
//                   sh 'make integration-test'
//                   sh 'make clean-cluster'
//               }
//             }
//           }
//         }
//
//         stage("AKS Integration Test") {
//           agent {
//             label "aks"
//           }
//           options {
//             timeout(time: 30, unit: 'MINUTES')
//           }
//           tools {
//             go 'Go 1.17'
//           }
//           environment {
//             GCP_CREDS = credentials("GCP_CREDS")
//             AKS_CLUSTER_NAME = "k8po-ci"
//           }
//           steps {
//             withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
//               sh './hack/jenkins/setup-for-integration-test.sh'
//               sh './hack/jenkins/install_docker_buildx.sh'
//               sh 'make semver-cli'
//               lock("integration-test-aks") {
//                 withCredentials([file(credentialsId: 'aks-kube-config', variable: 'KUBECONFIG')]) {
//                   sh 'kubectl config use k8po-ci'
//                   sh 'make clean-cluster'
//                   sh 'make integration-test'
//                   sh 'make clean-cluster'
//                 }
//               }
//             }
//           }
//         }
      }
    }

  }
  post {
    regression {
      slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "CI OPERATOR BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
    fixed {
      slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "CI OPERATOR BUILD FIXED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
  }
}
