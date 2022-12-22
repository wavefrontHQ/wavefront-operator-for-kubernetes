pipeline {
  agent any

  tools {
    go 'Go 1.17'
  }


  environment {
    PATH = "${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
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
    stage("Test Go Code") {
      agent {
        label "golang"
      }
      steps {
        sh 'make checkfmt vet test'
        sh 'make linux-golangci-lint'
        sh 'make golangci-lint'
      }
    }
    stage("Setup For Publish") {
      agent {
        label "integration"
      }
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
      }
      steps {
        sh './hack/jenkins/setup-for-integration-test.sh'
        sh './hack/jenkins/install_docker_buildx.sh'
        sh 'make semver-cli'
      }
    }
    stage("Publish") {
      agent {
        label "integration"
      }
      environment {
        RELEASE_TYPE = "alpha"
      }
      steps {
        sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
        sh 'make docker-xplatform-build'
      }
    }
    stage("Update RC branch") {
      environment {
        RELEASE_TYPE = "alpha"
        TOKEN = credentials('GITHUB_TOKEN')
      }
      steps {
        sh './hack/jenkins/create-rc-ci.sh'
        script {
          env.OPERATOR_YAML_RC_SHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
        }
      }
    }

    stage("Run Integration Tests") {
      environment {
        OPERATOR_YAML_TYPE="rc"
      }

      parallel {
        stage("GKE") {
          agent {
            label "integration"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
            GCP_ZONE="a"
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
          }
          steps {
            sh './hack/jenkins/setup-for-integration-test.sh'
            sh './hack/jenkins/install_docker_buildx.sh'
            sh 'make semver-cli'
            lock("integration-test-gke") {
                sh 'make gke-connect-to-cluster'
                sh 'make clean-cluster'
                sh 'make integration-test'
                sh 'make clean-cluster'
                sh 'KUSTOMIZATION_TYPE=custom NS=custom-namespace INTEGRATION_TEST_ARGS="-r advanced" make integration-test'
                sh 'make clean-cluster'
            }
          }
        }

        stage("EKS") {
          agent {
            label "integration"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
            AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
          }
          steps {
            sh './hack/jenkins/setup-for-integration-test.sh'
            sh './hack/jenkins/install_docker_buildx.sh'
            sh 'make semver-cli'
            lock("integration-test-eks") {
                sh 'make target-eks'
                sh 'make clean-cluster'
                sh 'make integration-test'
                sh 'make clean-cluster'
            }
          }
        }

        stage("AKS") {
          agent {
            label "integration"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            AKS_CLUSTER_NAME = "k8po-ci"
          }
          steps {
            sh './hack/jenkins/setup-for-integration-test.sh'
            sh './hack/jenkins/install_docker_buildx.sh'
            sh 'make semver-cli'
            lock("integration-test-aks") {
              withCredentials([file(credentialsId: 'aks-kube-config', variable: 'KUBECONFIG')]) {
                sh 'kubectl config use k8po-ci'
                sh 'make clean-cluster'
                sh 'make integration-test'
                sh 'make clean-cluster'
              }
            }
          }
        }
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
