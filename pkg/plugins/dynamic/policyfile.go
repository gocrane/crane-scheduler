package dynamic

import (
	"fmt"
	"io/ioutil"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy/scheme"
)

func LoadPolicyFromFile(file string) (*policy.DynamicSchedulerPolicy, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return loadPolicy(data)
}

func loadPolicy(data []byte) (*policy.DynamicSchedulerPolicy, error) {
	// The UniversalDecoder runs defaulting and returns the internal type by default.
	obj, gvk, err := scheme.Codecs.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	if policyObj, ok := obj.(*policy.DynamicSchedulerPolicy); ok {
		policyObj.TypeMeta.APIVersion = gvk.GroupVersion().String()
		return policyObj, nil
	}

	return nil, fmt.Errorf("couldn't decode as DynamicSchedulerPolicy, got %s: ", gvk)
}
