package balance

// func TestConsistentHashing(t *testing.T) {
// 	hash := NewConsistentHashing(10, nil)

// 	services := []*registry.Service{
// 		{
// 			Name:     "host1",
// 			Version:  "v1",
// 			Addr:     "localhost:8080",
// 			Metadata: make(map[string]string),
// 			Weight:   3,
// 		},
// 		{
// 			Name:     "host2",
// 			Version:  "v1",
// 			Addr:     "localhost:8081",
// 			Metadata: make(map[string]string),
// 			Weight:   2,
// 		},
// 		{
// 			Name:     "host3",
// 			Version:  "v1",
// 			Addr:     "localhost:8082",
// 			Metadata: make(map[string]string),
// 			Weight:   1,
// 		},
// 	}

// 	hash.Add(services...)

// 	testCases := map[string]string{
// 		"host1": "localhost:8080",
// 		"host2": "localhost:8081",
// 		"host3": "localhost:8082",
// 	}

// 	for k, v := range testCases {
// 		if res, _ := hash.Pick(k); res.Addr != v {
// 			t.Errorf("Asking for %s, get %s", v, res.Addr)
// 		}
// 	}

// 	hash.Add(&registry.Service{
// 		Name: "host4",
// 		Addr: "localhost:8083",
// 	})

// 	for k, v := range testCases {
// 		if res, _ := hash.Pick(k); res.Addr != v {
// 			t.Errorf("Asking for %s, get %s", v, res.Addr)
// 		}
// 	}

// 	hash.Remove(&registry.Service{
// 		Name: "host1",
// 		Addr: "localhost:8080",
// 	})

// 	for k, v := range testCases {
// 		if res, _ := hash.Pick(k); res.Addr != v {
// 			t.Errorf("Asking for %s, get %s", v, res.Addr)
// 		}
// 	}
// }
