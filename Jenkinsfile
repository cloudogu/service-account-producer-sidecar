#!groovy
@Library(['github.com/cloudogu/ces-build-lib@4.1.0']) _
import com.cloudogu.ces.cesbuildlib.*

Git git = new Git(this, "cesmarvin")
git.committerName = 'cesmarvin'
git.committerEmail = 'cesmarvin@cloudogu.com'
gitflow = new GitFlow(this, git)
github = new GitHub(this, git)
String productionReleaseBranch = "main"
Docker docker = new Docker(this)

goVersion = "1.26.4"

node('docker') {
    timestamps {
        stage('Checkout') {
            checkout scm
            make 'clean'
        }

        new Docker(this)
                .image("golang:${goVersion}")
                .mountJenkinsUser()
                .inside("--volume ${WORKSPACE}:/workdir -w /workdir") {
                    stage('Build') {
                        make 'compile-ci'
                    }

                    stage('Unit Tests') {
                        make 'unit-test'
                        junit allowEmptyResults: true, testResults: 'target/unit-tests/*.xml'
                    }

                    stage('Static Analysis') {
                        make 'static-analysis-ci'
                    }
                }

        String imageName = getImageName()
        String imageVersion = getImageVersion()

        if (gitflow.isReleaseBranch()) {
            Changelog changelog = new Changelog(this)

            stage('Finish Release') {
                gitflow.finishRelease("v${imageVersion}", productionReleaseBranch)
            }

            stage('Add Github-Release') {
                github.createReleaseWithChangelog("v${imageVersion}", changelog, productionReleaseBranch)
            }
        }

        def dockerImage
        stage('Build Image') {
            String branchName = git.getSimpleBranchName()
            if (gitflow.isReleaseBranch()) {
                imageName += ":${imageVersion}"
            } else if (branchName == 'main') {
                imageName += ":latest"
            } else if (branchName == 'develop') {
                imageName += ":develop"
            } else {
                imageName += ":tmp"
            }
            dockerImage = docker.build(imageName, ".")
        }
        stage('Deploy Image') {
            docker.withRegistry("https://registry.cloudogu.com", 'cesmarvin-setup') {
                dockerImage.push()
            }
        }
    }
}

String getImageName() {
    sh(returnStdout: true,
            script: "make name 2>/dev/null ")
            .trim()
}

String getImageVersion() {
    sh(returnStdout: true,
            script: "make version 2>/dev/null")
            .trim()
}

void make(String makeArgs) {
    sh "make ${makeArgs}"
}
