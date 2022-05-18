pipeline {
  agent any

  environment {
    GITHUB_CREDS_PSW = credentials("GITHUB_TOKEN")
  }

  stages {
    stage("Test with Go 1.17") {
      tools {
        go 'Go 1.17'
      }
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
          sh 'make checkfmt vet test'
        }
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
        DOCKER_IMAGE = "kubernetes-collector-snapshot"
      }
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
          sh 'make docker-build'
          sh 'make semver-cli'
          sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
          sh 'HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make publish'
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