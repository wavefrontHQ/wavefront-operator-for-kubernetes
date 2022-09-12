pipeline {
  agent any

  environment {
    GITHUB_CREDS_PSW = credentials("GITHUB_TOKEN")
  }

  stages {
    stage("Test Go Code") {
      tools {
        go 'Go 1.17'
      }
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
          sh 'make checkfmt vet test'
        }
      }
    }
    stage("Setup For Publish") {
      tools {
        go 'Go 1.17'
      }
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
        GKE_CLUSTER_NAME = "k8po-jenkins-ci"
        WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
        VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
        PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
        DOCKER_IMAGE = "kubernetes-operator-snapshot"
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
        VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
        HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
        PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
        DOCKER_IMAGE = "kubernetes-operator-snapshot"
      }
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
          sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
          sh 'HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make docker-xplatform-build'
        }
      }
    }
    stage("Run Integration Tests") {
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
          }
          environment {
            GKE_CLUSTER_NAME = "k8po-jenkins-ci"
            GCP_CREDS = credentials("GCP_CREDS")
            VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
            PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
            HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
            DOCKER_IMAGE = "kubernetes-operator-snapshot"
            WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
            WAVEFRONT_LOGGING_TOKEN = credentials("WAVEFRONT_TOKEN_SPRINGLOGS")
            GCP_PROJECT = "wavefront-gcp-dev"
          }
          steps {
            withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
              sh './hack/jenkins/setup-for-integration-test.sh'
              sh './hack/jenkins/install_docker_buildx.sh'
              sh 'make semver-cli'
              lock("integration-test-gke") {
                sh 'make gke-connect-to-cluster'
                sh 'make integration-test-ci'
                sh 'make undeploy'
              }
            }
          }
        }

        stage("AKS Integration Test") {
          agent {
            label "aks"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          tools {
            go 'Go 1.17'
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GKE_CLUSTER_NAME = "k8po-jenkins-ci"
            AKS_CLUSTER_NAME = "k8po-ci"
            VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
            PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
            HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
            DOCKER_IMAGE = "kubernetes-operator-snapshot"
            WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
            WAVEFRONT_LOGGING_TOKEN = credentials("WAVEFRONT_TOKEN_SPRINGLOGS")
          }
          steps {
            withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
              sh './hack/jenkins/setup-for-integration-test.sh'
              sh './hack/jenkins/install_docker_buildx.sh'
              sh 'make semver-cli'
              lock("integration-test-aks") {
                withCredentials([file(credentialsId: 'aks-kube-config', variable: 'KUBECONFIG')]) {
                  sh 'kubectl config use k8po-ci'
                  sh 'make integration-test-ci'
                  sh 'make undeploy'
                }
              }
            }
          }
        }
      }
    }
    stage("Update RC branch") {
      tools {
        go 'Go 1.17'
      }
      environment {
        RELEASE_TYPE = "alpha"
        VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
        HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
        PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
        DOCKER_IMAGE = "kubernetes-operator-snapshot"
        TOKEN = credentials('GITHUB_TOKEN')
        GIT_BRANCH = "rc${VERSION_POSTFIX}"
      }
      steps {
        withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          script{
            if (env.BRANCH_NAME == 'main') {
              sh './hack/jenkins/create-rc-ci.sh'
            }
          }
        }
      }
    }
  }
  post {
    // Notify only on null->failure or success->failure or failure->success
    failure {
      script {
        if(currentBuild.previousBuild == null) {
          slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "CI OPERATOR BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
        }
      }
    }
    regression {
      slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "CI OPERATOR BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
    fixed {
      slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "CI OPERATOR BUILD FIXED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
  }
}
