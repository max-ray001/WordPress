pipeline {
    agent { label 'upbound-gce' }

    parameters {
        string(name: 'version', defaultValue: '', description: 'The version you are tagging. For example: v0.4.0. If left unspecified, the build will generate one for you.')
        string(name: 'commit', defaultValue: '', description: 'Optional commit hash to tag. For example: 56b65dba917e50132b0a540ae6ff4c5bbfda2db6. If empty, the latest commit hash will be used.')
    }

    options {
        disableConcurrentBuilds()
        timestamps()
    }

    environment {
        GITHUB_UPBOUND_BOT = credentials('github-upbound-jenkins')
    }

    stages {

        stage('Prepare') {
            steps {
                 // github credentials are not setup to push over https in jenkins. add the github token to the url
                sh "git config remote.origin.url https://${GITHUB_UPBOUND_BOT_USR}:${GITHUB_UPBOUND_BOT_PSW}@\$(git config --get remote.origin.url | sed -e 's/https:\\/\\///')"
                sh 'git config user.name "upbound-bot"'
                sh 'git config user.email "info@crossplane.io"'
                sh 'echo "machine github.com login upbound-bot password $GITHUB_UPBOUND_BOT" > ~/.netrc'
            }
        }

        stage('Tag Release') {
            steps {
                // The first few lines set the version - it uses the passed version, or
                // generates one.
                // If the commit is not passed, it'll be empty, which means git will tag the head
                // of the current branch. Most of the time, the default behavior will be used.
                //
                // For the push, we're assuming the remote is named "origin", but this is the convention,
                // so it's a safe assumption for this situation.
                sh """VERSION='${params.version}'
                      VERSION=\${VERSION:-\$( git describe --tags --dirty --always )}
                      git tag -f -m "release \${VERSION}" \${VERSION} ${params.commit}
                      git push origin \${VERSION}
                """
            }
        }
    }

    post {
        always {
            deleteDir()
        }
    }
}
