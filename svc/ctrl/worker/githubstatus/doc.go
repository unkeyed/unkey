// Package githubstatus reports deployment state to GitHub. Alongside the legacy
// status handlers, the notification integration uses separate deployment-keyed
// and PR-keyed objects to order reports and shared comment rows. GitHub write
// failures can leave unknown outcomes rather than prove that no write occurred.
package githubstatus
