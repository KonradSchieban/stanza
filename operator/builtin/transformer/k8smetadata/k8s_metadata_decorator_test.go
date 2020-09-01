package k8smetadata

import (
	"context"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMetadataCache(t *testing.T) {
	m := MetadataCache{}
	entry := MetadataCacheEntry{
		Labels: map[string]string{
			"label1": "value1",
		},
		Annotations: map[string]string{
			"annotation2": "value2",
		},
	}

	m.Store("testkey", entry)
	loaded, ok := m.Load("testkey")
	require.True(t, ok)
	require.Equal(t, entry, loaded)
}

func basicConfig() *K8sMetadataDecoratorConfig {
	cfg := NewK8sMetadataDecoratorConfig("testoperator")
	cfg.OutputIDs = []string{"mock"}
	return cfg
}

func TestK8sMetadataDecoratorBuildDefault(t *testing.T) {
	cfg := basicConfig()

	expected := &K8sMetadataDecorator{
		TransformerOperator: helper.TransformerOperator{
			WriterOperator: helper.WriterOperator{
				BasicOperator: helper.BasicOperator{
					OperatorID:   "testoperator",
					OperatorType: "k8s_metadata_decorator",
				},
				OutputIDs: []string{"mock"},
			},
			OnError: "send",
		},
		podNameField:   entry.NewResourceField("k8s.pod.name"),
		namespaceField: entry.NewResourceField("k8s.namespace.name"),
		cacheTTL:       10 * time.Minute,
		timeout:        10 * time.Second,
	}

	operator, err := cfg.Build(testutil.NewBuildContext(t))
	operator.(*K8sMetadataDecorator).SugaredLogger = nil
	require.NoError(t, err)
	require.Equal(t, expected, operator)

}

func TestK8sMetadataDecoratorCachedMetadata(t *testing.T) {

	cfg := basicConfig()
	pg, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)

	mockOutput := testutil.NewMockOperator("mock")
	pg.SetOutputs([]operator.Operator{mockOutput})

	// Preload cache so we don't hit the network
	k8s := pg.(*K8sMetadataDecorator)
	k8s.namespaceCache.Store("testnamespace", MetadataCacheEntry{
		ExpirationTime: time.Now().Add(time.Hour),
		Labels: map[string]string{
			"label1": "lab1",
		},
		Annotations: map[string]string{
			"annotation1": "ann1",
		},
	})

	k8s.podCache.Store("testnamespace:testpodname", MetadataCacheEntry{
		ExpirationTime: time.Now().Add(time.Hour),
		Labels: map[string]string{
			"podlabel1": "podlab1",
		},
		Annotations: map[string]string{
			"podannotation1": "podann1",
		},
	})

	expected := entry.Entry{
		Labels: map[string]string{
			"k8s-pod/podlabel1":                 "podlab1",
			"k8s-ns/label1":                     "lab1",
			"k8s-pod-annotation/podannotation1": "podann1",
			"k8s-ns-annotation/annotation1":     "ann1",
		},
	}

	mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		entry := args.Get(1).(*entry.Entry)
		require.Equal(t, expected.Labels, entry.Labels)
	}).Return(nil)

	e := &entry.Entry{
		Resource: map[string]string{
			"k8s.pod.name":       "testpodname",
			"k8s.namespace.name": "testnamespace",
		},
	}
	err = pg.Process(context.Background(), e)
	require.NoError(t, err)
}
