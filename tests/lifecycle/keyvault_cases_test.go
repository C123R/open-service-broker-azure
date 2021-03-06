// +build !unit

package lifecycle

var keyvaultTestCases = []serviceLifecycleTestCase{
	{
		group:     "keyvault",
		name:      "keyvault",
		serviceID: "d90c881e-c9bb-4e07-a87b-fcfe87e03276",
		planID:    "3577ee4a-75fc-44b3-b354-9d33d52ef486",
		location:  "southcentralus",
		provisioningParameters: map[string]interface{}{
			"objectId":     "6a74d229-e927-42c5-b6e8-8f5c095cfba8",
			"clientId":     "test",
			"clientSecret": "test",
		},
	},
}
