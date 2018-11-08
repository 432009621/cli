package v6

import (
	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/command"
)

//go:generate counterfeiter . DomainsActor

type DomainsActor interface {
	GetDomains(orgGUID string) ([]v2action.Domain, v2action.Warnings, error)
}

type DomainsCommand struct {
	usage           interface{} `usage:"CF_NAME domains"`
	relatedCommands interface{} `related_commands:"router-groups, create-route, routes"`

	UI          command.UI
	Config      command.Config
	SharedActor command.SharedActor
	Actor       DomainsActor
}

func (cmd *DomainsCommand) Setup(config command.Config, ui command.UI) error {
	cmd.Config = config
	cmd.UI = ui
	cmd.SharedActor = sharedaction.NewActor(config)
	return nil
}

func (cmd DomainsCommand) Execute(args []string) error {
	err := cmd.SharedActor.CheckTarget(true, false)

	if err != nil {
		return err
	}

	org := cmd.Config.TargetedOrganization()

	user, err := cmd.Config.CurrentUser()

	if err != nil {
		return err
	}

	_, _, err = cmd.Actor.GetDomains(org.GUID)

	if err != nil {
		return err
	}

	cmd.UI.DisplayTextWithFlavor("Getting domains in org {{.CurrentOrg}} as {{.CurrentUser}}...", map[string]interface{}{
		"CurrentUser": user.Name,
		"CurrentOrg":  org.Name,
	})

	return err
}
