pipeline {
  agent any

  tools {
    go 'Go 1.17'
  }

  environment {
    RELEASE_TYPE = 'release'
    RC_NUMBER = "1"
    BUMP_COMPONENT = "${params.BUMP_COMPONENT}"
    COLLECTOR_VERSION = "${params.COLLECTOR_VERSION}"
    GIT_BRANCH = getCurrentBranchName()
    GIT_CREDENTIAL_ID = 'wf-jenkins-github'
    TOKEN = credentials('GITHUB_TOKEN')
  }

  stages {
    stage("Setup tools") {
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
          sh './hack/jenkins/install_docker_buildx.sh'
          sh 'make semver-cli'
        }
      }
    }
    stage("Create Bump Version Branch") {
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]){
          sh 'git config --global user.email "svc.wf-jenkins@vmware.com"'
          sh 'git config --global user.name "svc.wf-jenkins"'
          sh 'git remote set-url origin https://${TOKEN}@github.com/wavefronthq/wavefront-operator-for-kubernetes.git'
          sh './hack/jenkins/create-bump-version-branch.sh -s "${BUMP_COMPONENT}" -c "${COLLECTOR_VERSION}"'
        }
      }
    }
    stage("Publish RC Release") {
      environment {
        HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
        PREFIX = 'projects.registry.vmware.com/tanzu_observability_keights_saas'
        DOCKER_IMAGE = 'kubernetes-operator-snapshot'
      }
      steps {
        script {
          env.READ_VERSION = readFile('./release/OPERATOR_VERSION').trim()
          env.VERSION = "${env.READ_VERSION}-rc-1"
        }
        sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
        sh 'HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make docker-xplatform-build generate-kubernetes-yaml'
      }
    }
    // deploy to GKE and run manual tests
    // now we have confidence in the validity of our RC release
    stage("Deploy and Test") {
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
        GKE_CLUSTER_NAME = "k8po-jenkins-ci"
        WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
        WF_CLUSTER = 'nimba'
      }
      steps {
        script {
          env.READ_VERSION = readFile('./release/OPERATOR_VERSION').trim()
          env.VERSION = "${env.READ_VERSION}-rc-1"
        }
        withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          lock("integration-test-gke") {
            sh './hack/jenkins/setup-for-integration-test.sh'
            sh 'make gke-connect-to-cluster'
            sh 'make integration-test-ci'
            sh 'make undeploy'
          }
        }
      }
    }
    stage("Publish GA Harbor Image") {
      environment {
        HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
        PREFIX = 'projects.registry.vmware.com/tanzu_observability_keights_saas'
        DOCKER_IMAGE = 'kubernetes-operator-release-like'
      }
      steps {
        script {
          env.VERSION = readFile('./release/VERSION').trim()
          env.RC_VERSION = "${env.VERSION}-rc-1"
        }
        sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
        sh 'docker pull $PREFIX/$DOCKER_IMAGE:$RC_VERSION'
        sh 'docker tag $PREFIX/$DOCKER_IMAGE:$RC_VERSION $PREFIX/$DOCKER_IMAGE:$VERSION'
        sh 'docker push $PREFIX/$DOCKER_IMAGE:$VERSION'
      }
    }
//     stage("Create and Merge Bump Version Pull Request") {
//       steps {
//         sh './hack/jenkins/create-and-merge-pull-request.sh'
//       }
//     }
//     stage("Github Release") {
//       environment {
//         GITHUB_CREDS_PSW = credentials("GITHUB_TOKEN")
//       }
//       steps {
//         sh './hack/jenkins/generate_github_release.sh'
//       }
//     }
  }

//   post {
//     // Notify only on null->failure or success->failure or any->success
//     failure {
//       script {
//         if(currentBuild.previousBuild == null) {
//           slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
//         }
//       }
//     }
//     regression {
//       slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
//     }
//     success {
//       script {
//         BUILD_VERSION = readFile('./release/VERSION').trim()
//         slackSend (channel: '#tobs-k8s-assist', color: '#008000', message: "Success!! `wavefront-collector-for-kubernetes:v${BUILD_VERSION}` released!")
//       }
//     }
//   }
}

def getCurrentBranchName() {
  return env.BRANCH_NAME.split("/")[1]
}