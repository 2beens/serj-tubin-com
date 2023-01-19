package aerospike

import (
	"fmt"

	as "github.com/aerospike/aerospike-client-go"
)

type AeroBinMap map[string]interface{}

func RecordSet2AeroBinMaps(recordSet *as.Recordset) ([]AeroBinMap, error) {
	var binMap []AeroBinMap
	for rec := range recordSet.Results() {
		if rec.Err != nil {
			return nil, fmt.Errorf("query by range, record error: %s", rec.Err)
		}
		aeroBin := make(map[string]interface{})
		for k, v := range rec.Record.Bins {
			aeroBin[k] = v
		}
		binMap = append(binMap, aeroBin)
	}

	return binMap, nil
}
