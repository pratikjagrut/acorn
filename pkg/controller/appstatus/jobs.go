package appstatus

import (
	"fmt"
	"strconv"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	batchv1 "k8s.io/api/batch/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func (a *appStatusRenderer) readJobs() error {
	var (
		existingStatus = a.app.Status.AppStatus.Jobs
	)

	// reset state
	a.app.Status.AppStatus.Jobs = make(map[string]v1.JobStatus, len(a.app.Status.AppSpec.Jobs))

	summary, err := a.getReplicasSummary(labels.AcornJobName)
	if err != nil {
		return err
	}

	for jobName := range a.app.Status.AppSpec.Jobs {
		c := v1.JobStatus{
			CreateEventSucceeded: existingStatus[jobName].CreateEventSucceeded,
			Skipped:              existingStatus[jobName].Skipped,
			ExpressionErrors:     existingStatus[jobName].ExpressionErrors,
			Dependencies:         existingStatus[jobName].Dependencies,
		}
		summary := summary[jobName]

		c.Defined = ports.IsLinked(a.app, jobName)
		c.LinkOverride = ports.LinkService(a.app, jobName)
		c.TransitioningMessages = append(c.TransitioningMessages, summary.TransitioningMessages...)
		c.ErrorMessages = append(c.ErrorMessages, summary.ErrorMessages...)
		c.RunningCount = summary.RunningCount

		if c.Skipped {
			c.Ready = true
			c.UpToDate = true
			c.Defined = true
			c.ErrorCount = 0
			c.RunningCount = 0
			c.Dependencies = nil
			a.app.Status.AppStatus.Jobs[jobName] = c
			continue
		}

		var job batchv1.Job
		err := a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, jobName), &job)
		if apierror.IsNotFound(err) {
			var cronJob batchv1.CronJob
			err := a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, jobName), &cronJob)
			if apierror.IsNotFound(err) {
				// do nothing
			} else if err != nil {
				return err
			} else {
				c.Defined = true
				c.UpToDate = cronJob.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation))
				c.RunningCount = len(cronJob.Status.Active)
				if cronJob.Status.LastSuccessfulTime != nil {
					c.CreateEventSucceeded = true
					c.Ready = c.UpToDate
				}
			}
		} else if err != nil {
			return err
		} else {
			c.Defined = true
			c.UpToDate = job.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation))
			if job.Status.Succeeded > 0 {
				c.CreateEventSucceeded = true
				c.Ready = c.UpToDate
			} else if job.Status.Failed > 0 {
				c.ErrorCount = int(job.Status.Failed)
			} else if job.Status.Active > 0 && c.RunningCount == 0 {
				c.RunningCount = int(job.Status.Active)
			}
		}

		if c.RunningCount > 0 {
			c.TransitioningMessages = append(c.TransitioningMessages, "running")
		} else if c.ErrorCount > 0 {
			c.ErrorMessages = append(c.ErrorMessages, fmt.Sprintf("%d failed attempts", c.ErrorCount))
		}

		if c.LinkOverride != "" {
			var err error
			c.UpToDate = true
			c.Ready, c.Defined, err = a.isServiceReady(jobName)
			if err != nil {
				return err
			}
			if c.Ready {
				c.CreateEventSucceeded = true
			}
		}

		for _, entry := range typed.Sorted(c.Dependencies) {
			depName, dep := entry.Key, entry.Value
			if !dep.Ready {
				c.Ready = false
				msg := fmt.Sprintf("%s %s dependency is not ready", dep.DependencyType, depName)
				if dep.Missing {
					msg = fmt.Sprintf("%s %s dependency is missing", dep.DependencyType, depName)
				}
				c.TransitioningMessages = append(c.TransitioningMessages, msg)
			}
		}

		addExpressionErrors(&c.CommonStatus, c.ExpressionErrors)

		a.app.Status.AppStatus.Jobs[jobName] = c
	}

	return nil
}

func addExpressionErrors(status *v1.CommonStatus, expressionErrors []v1.ExpressionError) {
	missing := map[string]v1.DependencyType{}
	for _, ee := range expressionErrors {
		status.Ready = false
		if ee.DependencyNotFound != nil {
			missing[ee.DependencyNotFound.Name] = ee.DependencyNotFound.DependencyType
		} else if ee.Error != "" {
			if ee.Expression == "" {
				status.ErrorMessages = append(status.ErrorMessages, ee.Error)
			} else {
				status.ErrorMessages = append(status.ErrorMessages, fmt.Sprintf("[%s]: %s", ee.Expression, ee.Error))
			}
		}
	}

	for _, entry := range typed.Sorted(missing) {
		status.TransitioningMessages = append(status.TransitioningMessages, fmt.Sprintf("%s [%s] missing", entry.Value, entry.Key))
	}
}

func (a *appStatusRenderer) isJobReady(jobName string) (ready bool, err error) {
	var jobDep batchv1.Job
	err = a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, jobName), &jobDep)
	if apierror.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		// if err just return it as not ready
		return false, err
	}

	if jobDep.Annotations[labels.AcornAppGeneration] != strconv.Itoa(int(a.app.Generation)) ||
		jobDep.Status.Succeeded != 1 {
		return false, nil
	}

	return true, nil
}
