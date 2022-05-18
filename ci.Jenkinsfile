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
          sh 'make fmt vet test'
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