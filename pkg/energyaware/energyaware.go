package energyaware

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	framework "k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/scheduler-plugins/apis/config"
)

const Name = "EnergyAware"

var _ framework.ScorePlugin = &EnergyAware{}

type EnergyAware struct {
	handle     framework.Handle
	prometheus *PrometheusHandle
}

func (pl *EnergyAware) Name() string {
	return Name
}

func (e *EnergyAware) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := e.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("[EnergyAware] failed to get node info for %s: %v", nodeName, err))
	}
	internalIP, err := getInternalIP(nodeInfo.Node())

	if err != nil {
		return 0, framework.NewStatus(framework.Error, err.Error())
	}
	// Now pass internalIP instead of nodeName to Prometheus query
	nodeBandwidth, err := e.prometheus.GetNodeBandwidthMeasure(internalIP)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, err.Error())
	}

	klog.Infof("[EnergyAware] node '%s' (%s) bandwidth: %s", nodeName, internalIP, nodeBandwidth.Value)
	return int64(nodeBandwidth.Value), nil
}

func (e *EnergyAware) ScoreExtensions() framework.ScoreExtensions {
	return e
}

func (e *EnergyAware) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	var higherScore int64
	for _, node := range scores {
		if node.Score > higherScore {
			higherScore = node.Score
		}
	}

	for i, node := range scores {
		scores[i].Score = framework.MaxNodeScore - (node.Score * framework.MaxNodeScore / higherScore)
	}

	klog.Infof("[EnergyAware] Nodes final score: %v", scores)
	return nil
}

func New(_ context.Context, obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	args, ok := obj.(*config.EnergyAwareArgs)
	if !ok {
		return nil, fmt.Errorf("[EnergyAware] want args to be of type EnergyAwareArgs, got %T", obj)
	}

	klog.Infof("[EnergyAware] args received. NetworkInterface: %s; TimeRangeInMinutes: %d, Address: %s", args.NetworkInterface, args.TimeRangeInMinutes, args.Address)

	return &EnergyAware{
		handle:     h,
		prometheus: NewPrometheus(args.Address, args.NetworkInterface, time.Minute*time.Duration(args.TimeRangeInMinutes)),
	}, nil
}
