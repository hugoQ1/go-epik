package main

import (
	"fmt"
	"testing"

	"github.com/EpiK-Protocol/go-epik/cmd/epik-expressman/workspace"
	"github.com/stretchr/testify/require"
)

func TestUploadWorkspace(t *testing.T) {
	ws, err := workspace.LoadUploadWorkspace("./testdata")
	require.NoError(t, err)
	fmt.Println("done: ", ws.Done())

	err = ws.ForEach(func(df workspace.DataFile) error {
		fmt.Println(">>", df.Filename, df.OnchainID)
		return nil
	})
	require.NoError(t, err)

	err = ws.SetOnchainID("yizhu_menjizhen_20210323_115605_T_2.sql", "dfafewfwsfasf")
	require.NoError(t, err)

	require.NoError(t, ws.Finalize())
}
