/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package priority

import (
	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kube-batch/pkg/scheduler/api"
	"github.com/kubernetes-sigs/kube-batch/pkg/scheduler/framework"

	"k8s.io/api/core/v1"
)

type priorityPlugin struct {
	args *framework.PluginArgs
}

func New(args *framework.PluginArgs) framework.Plugin {
	return &priorityPlugin{
		args: args,
	}
}

func (pp *priorityPlugin) OnSessionOpen(ssn *framework.Session) {
	taskOrderFn := func(l interface{}, r interface{}) int {
		lv := l.(*api.TaskInfo)
		rv := r.(*api.TaskInfo)

		// TODO quick fix, enhance it later by other option.
		if res := compareQoS(lv.Pod, rv.Pod); res != 0 {
			return res
		}

		glog.V(3).Infof("Priority TaskOrder: <%v/%v> prority is %v, <%v/%v> priority is %v",
			lv.Namespace, lv.Name, lv.Priority, rv.Namespace, rv.Name, rv.Priority)

		if lv.Priority == rv.Priority {
			return 0
		}

		if lv.Priority > rv.Priority {
			return -1
		}

		return 1
	}

	// Add Task Order function
	if pp.args.TaskOrderFnEnabled {
		ssn.AddTaskOrderFn(taskOrderFn)
	}

	jobOrderFn := func(l, r interface{}) int {
		lv := l.(*api.JobInfo)
		rv := r.(*api.JobInfo)

		glog.V(3).Infof("Priority JobOrderFn: <%v/%v> is ready: %d, <%v/%v> is ready: %d",
			lv.Namespace, lv.Name, lv.Priority, rv.Namespace, rv.Name, rv.Priority)

		if lv.Priority > rv.Priority {
			return -1
		}

		if lv.Priority < rv.Priority {
			return 1
		}

		return 0
	}

	if pp.args.JobOrderFnEnabled {
		// Add Job Order function
		ssn.AddJobOrderFn(jobOrderFn)
	}
}

func (pp *priorityPlugin) OnSessionClose(ssn *framework.Session) {}

// make BestEffort > Burstable/Guarantee
func compareQoS(l, r *v1.Pod) int {
	lq := l.Status.QOSClass
	rq := r.Status.QOSClass

	glog.V(3).Infof("Priority TaskOrder: <%v/%v> QoS is %v, <%v/%v> QoS is %v",
		l.Namespace, l.Name, lq, r.Namespace, r.Name, rq)

	if lq == rq {
		return 0
	}

	// BestEffort > Burstable/Guarantee
	if lq == v1.PodQOSBestEffort {
		return -1
	}

	// Burstable/Guarantee < BestEffort
	if rq == v1.PodQOSBestEffort {
		return 1
	}

	return 0
}
