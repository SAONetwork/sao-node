package rule_engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	TestCar struct {
		SpeedUp        bool
		Speed          uint
		MaxSpeed       uint
		SpeedIncrement uint
	}

	DistanceRecord struct {
		TotalDistance uint
	}
)

func TestRuleEngine(t *testing.T) {
	svc := NewRuleEngineSvc()

	rule := `{
		"name": "SpeedUp",
		"desc": "When testcar is speeding up we increase the speed.",
		"salience": 10,
		"when": "TestCar.SpeedUp == true && TestCar.Speed < TestCar.MaxSpeed",
		"then": [
			"TestCar.Speed = TestCar.Speed + TestCar.SpeedIncrement",
			"TestCar.SpeedUp = TestCar.Speed != TestCar.MaxSpeed",
			"DistanceRecord.TotalDistance = DistanceRecord.TotalDistance + TestCar.Speed"
		]
	}`

	svc.AddRule("SpeedUp", []byte(rule))

	testCar := &TestCar{
		SpeedUp:        true,
		Speed:          20,
		MaxSpeed:       300,
		SpeedIncrement: 10,
	}

	distanceRecord := &DistanceRecord{
		TotalDistance: 0,
	}

	err := svc.AddFact("SpeedUpTest", "TestCar", testCar)
	require.NoError(t, err)

	err = svc.AddFact("SpeedUpTest", "DistanceRecord", distanceRecord)
	require.NoError(t, err)

	t.Logf("%v", testCar)
	t.Logf("%v", distanceRecord)

	err = svc.Execute("SpeedUp", "SpeedUpTest")
	require.NoError(t, err)

	t.Logf("%v", testCar)
	t.Logf("%v", distanceRecord)

	require.Equal(t, false, testCar.SpeedUp)
	require.Equal(t, uint(300), testCar.Speed)
	require.Equal(t, uint(0x120c), distanceRecord.TotalDistance)
}
