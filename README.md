# CD-webhook
This project functions based on webhook requests triggered by commits in a GitHub repository. Whenever a commit is 
created and if a webhook is properly configured to communicate with the server managed by this project, it commences
an integrity check of the webhook.

The main feature of this project lies in its ability to synchronize Kubernetes resources between a GitHub repository
and a local environment. If there are any Kubernetes resources present in the GitHub repository that are not yet installed
in the local environment, the system identifies this discrepancy. It then initiates an application process to integrate those
missing resources into the local Kubernetes cluster.

This not only automates the process of Kubernetes resource management but also ensures that the local environment is always
in sync with the latest version control changes from the GitHub repository.
