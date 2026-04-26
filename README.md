# infrared

AKA `ir`

_DISCLAIMER: Fully vibe coded_

This is a tool that helps us manage our infra more easily, at the present time the big benefit is setting up flux receiver webhooks on new repositories, but more tools will be added constantly to help with infra/devops work.

## Install and Use

### Prerequisites

- `kubectl` installed
- kubeconfig with enough permissions
- `gh` installed and authenticated with Github

Install `ir`:  
`go install github.com/and-fm/infrared/cmd/ir@main`

Then to run, simply execute `ir` in the terminal
