package v2action

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/api/cloudcontroller"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"
)

type ApplicationInstanceState ccv2.ApplicationInstanceState

type ApplicationInstance struct {
	// CPU is the instance's CPU utilization percentage.
	CPU float64

	// Details are arbitrary information about the instance.
	Details string

	// Disk is the instance's disk usage in bytes.
	Disk int

	// DiskQuota is the instance's allowed disk usage in bytes.
	DiskQuota int

	// ID is the instance ID.
	ID int

	// Memory is the instance's memory usage in bytes.
	Memory int

	// MemoryQuota is the instance's allowed memory usage in bytes.
	MemoryQuota int

	// Since is the Unix time stamp that represents the time the instance was
	// created.
	Since float64

	// State is the instance's state.
	State ApplicationInstanceState
}

// newApplicationInstance returns a pointer to a new ApplicationInstance.
func newApplicationInstance(id int) ApplicationInstance {
	return ApplicationInstance{ID: id}
}

func (instance ApplicationInstance) TimeSinceCreation() time.Time {
	return time.Unix(int64(instance.Since), 0)
}

func (instance *ApplicationInstance) setInstance(ccAppInstance ccv2.ApplicationInstance) {
	instance.Details = ccAppInstance.Details
	instance.Since = ccAppInstance.Since
	instance.State = ApplicationInstanceState(ccAppInstance.State)
}

func (instance *ApplicationInstance) setStats(ccAppStats ccv2.ApplicationInstanceStatus) {
	instance.CPU = ccAppStats.CPU
	instance.Disk = ccAppStats.Disk
	instance.DiskQuota = ccAppStats.DiskQuota
	instance.Memory = ccAppStats.Memory
	instance.MemoryQuota = ccAppStats.MemoryQuota
}

func (instance *ApplicationInstance) incomplete() {
	instance.Details = strings.TrimSpace(fmt.Sprintf("%s (%s)", instance.Details, "Unable to retrieve information"))
}

type applicationInstances []ApplicationInstance

func (a applicationInstances) Len() int               { return len(a) }
func (a applicationInstances) Swap(i int, j int)      { a[i], a[j] = a[j], a[i] }
func (a applicationInstances) Less(i int, j int) bool { return a[i].ID < a[j].ID }

// ApplicationInstancesNotFoundError is returned when the application does not
// have running instances.
type ApplicationInstancesNotFoundError struct {
	ApplicationGUID string
}

func (e ApplicationInstancesNotFoundError) Error() string {
	return fmt.Sprintf("Application instances '%s' not found.", e.ApplicationGUID)
}

func (actor Actor) GetApplicationInstancesByApplication(guid string) ([]ApplicationInstance, Warnings, error) {
	var allWarnings Warnings

	appInstanceStats, warnings, err := actor.CloudControllerClient.GetApplicationInstanceStatusesByApplication(guid)
	allWarnings = append(allWarnings, warnings...)

	switch err.(type) {
	case cloudcontroller.ResourceNotFoundError, ccv2.AppStoppedStatsError:
		return nil, allWarnings, ApplicationInstancesNotFoundError{ApplicationGUID: guid}
	case nil:
		// continue
	default:
		return nil, allWarnings, err
	}

	appInstances, warnings, err := actor.CloudControllerClient.GetApplicationInstancesByApplication(guid)
	allWarnings = append(allWarnings, warnings...)

	switch err.(type) {
	case ccv2.NotStagedError, ccv2.InstancesError:
		return nil, allWarnings, ApplicationInstancesNotFoundError{ApplicationGUID: guid}
	case nil:
		// continue
	default:
		return nil, allWarnings, err
	}

	returnedInstances := combineStatsAndInstances(appInstanceStats, appInstances)

	sort.Sort(returnedInstances)

	return returnedInstances, allWarnings, err
}

func combineStatsAndInstances(appInstanceStats map[int]ccv2.ApplicationInstanceStatus, appInstances map[int]ccv2.ApplicationInstance) applicationInstances {
	returnedInstances := applicationInstances{}
	seenStatuses := make(map[int]bool, len(appInstanceStats))

	for id, appInstanceStat := range appInstanceStats {
		seenStatuses[id] = true

		returnedInstance := newApplicationInstance(id)
		returnedInstance.setStats(appInstanceStat)

		if appInstance, found := appInstances[id]; found {
			returnedInstance.setInstance(appInstance)
		} else {
			returnedInstance.incomplete()
		}

		returnedInstances = append(returnedInstances, returnedInstance)
	}

	// add instances that are missing stats
	for index, appInstance := range appInstances {
		if _, found := seenStatuses[index]; !found {
			returnedInstance := newApplicationInstance(index)
			returnedInstance.setInstance(appInstance)
			returnedInstance.incomplete()

			returnedInstances = append(returnedInstances, returnedInstance)
		}
	}

	return returnedInstances
}