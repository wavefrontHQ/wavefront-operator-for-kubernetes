pipeline {
  agent any

  tools {
    go 'Go 1.17'
  }

  environment {
    BUMP_COMPONENT = "${params.BUMP_COMPONENT}"
    GIT_BRANCH = getCurrentBranchName()
    GIT_CREDENTIAL_ID = 'wf-jenkins-github'
    GITHUB_TOKEN = credentials('GITHUB_TOKEN')
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
          sh 'git remote set-url origin https://${GITHUB_TOKEN}@github.com/wavefronthq/wavefront-operator-for-kubernetes.git'
          sh './hack/jenkins/bump-version.sh -s "${BUMP_COMPONENT}"'
        }
      }
    }
    stage("Publish Image and Generate YAML") {
      environment {
        HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability-robot")
        PREFIX = 'projects.registry.vmware.com/tanzu_observability'
        DOCKER_IMAGE = 'kubernetes-operator'
      }
      steps {
        script {
          env.VERSION = readFile('./release/OPERATOR_VERSION').trim()
        }
        sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
        sh 'HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make docker-xplatform-build generate-kubernetes-yaml'
      }
    }
    // deploy to GKE and run manual tests
    stage("Deploy and Test") {
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
        GKE_CLUSTER_NAME = "k8po-jenkins-ci"
        WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
        WF_CLUSTER = 'nimba'
      }
      steps {
        script {
          env.VERSION = readFile('./release/OPERATOR_VERSION').trim()
        }
        withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          lock("integration-test-gke") {
            sh './hack/jenkins/setup-for-integration-test.sh'
            sh 'make gke-connect-to-cluster'
            sh 'make clean-cluster'
            sh './hack/test/deploy/deploy-local.sh -t $WAVEFRONT_TOKEN'
            sh './hack/test/run-e2e-tests.sh -t $WAVEFRONT_TOKEN'
            sh 'make clean-cluster'
          }
        }
      }
    }
    stage("Merge bumped versions") {
      steps {
        sh './hack/jenkins/merge-version-bump.sh'
      }
    }
    stage("Github Release") {
      steps {
        sh './hack/jenkins/generate-github-release.sh'
      }
    }
  }

  post {
    regression {
      slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
    success {
      script {
        BUILD_VERSION = readFile('./release/OPERATOR_VERSION').trim()
        slackSend (channel: '#tobs-k8s-assist', color: '#008000', message: "Success!! `wavefront-operator-for-kubernetes:v${BUILD_VERSION}` released!")
      }
    }
  }
}

def getCurrentBranchName() {
  return env.BRANCH_NAME.split("/")[1]
}