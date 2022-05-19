pipeline {
  agent any

  environment {
    GITHUB_CREDS_PSW = credentials("GITHUB_TOKEN")
  }

  stages {
//     stage("Test with Go 1.17") {
//       tools {
//         go 'Go 1.17'
//       }
//       steps {
//         withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
//           sh 'make checkfmt vet test'
//         }
//       }
//     }
    stage("Setup Integration Test") {
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
          sh 'make docker-build'
          sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
          sh 'HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make docker-push'
        }
      }
    }
    stage("GKE Integration Test") {
      options {
        timeout(time: 15, unit: 'MINUTES')
      }
      tools {
        go 'Go 1.17'
      }
      environment {
        GKE_CLUSTER_NAME = "k8po-jenkins-ci"
        VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
        PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
        HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
        DOCKER_IMAGE = "kubernetes-operator-snapshot"
        WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
        GCP_PROJECT = "wavefront-gcp-dev"
      }
      steps {
        withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          lock("integration-test-gke") {
            sh 'make gke-connect-to-cluster'
            sh 'make integration-test-ci'
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