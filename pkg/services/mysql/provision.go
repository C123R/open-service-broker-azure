package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-service-broker/pkg/azure"
	"github.com/Azure/azure-service-broker/pkg/generate"
	"github.com/Azure/azure-service-broker/pkg/service"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	uuid "github.com/satori/go.uuid"
)

func (m *module) ValidateProvisioningParameters(
	provisioningParameters service.ProvisioningParameters,
) error {
	pp, ok := provisioningParameters.(*ProvisioningParameters)
	if !ok {
		return errors.New(
			"error casting provisioningParameters as " +
				"*mysql.ProvisioningParameters",
		)
	}
	if !azure.IsValidLocation(pp.Location) {
		return service.NewValidationError(
			"location",
			fmt.Sprintf(`invalid location: "%s"`, pp.Location),
		)
	}
	return nil
}

func (m *module) GetProvisioner(string, string) (service.Provisioner, error) {
	return service.NewProvisioner(
		service.NewProvisioningStep("preProvision", m.preProvision),
		service.NewProvisioningStep("deployARMTemplate", m.deployARMTemplate),
	)
}

func (m *module) preProvision(
	ctx context.Context, // nolint: unparam
	instanceID string, // nolint: unparam
	serviceID string, // nolint: unparam
	planID string, // nolint: unparam
	provisioningContext service.ProvisioningContext,
	provisioningParameters service.ProvisioningParameters, // nolint: unparam
) (service.ProvisioningContext, error) {
	pc, ok := provisioningContext.(*mysqlProvisioningContext)
	if !ok {
		return nil, errors.New(
			"error casting provisioningContext as *mysqlProvisioningContext",
		)
	}
	pp, ok := provisioningParameters.(*ProvisioningParameters)
	if !ok {
		return nil, errors.New(
			"error casting provisioningParameters as " +
				"*mysql.ProvisioningParameters",
		)
	}
	if pp.ResourceGroup != "" {
		pc.ResourceGroupName = pp.ResourceGroup
	} else {
		pc.ResourceGroupName = uuid.NewV4().String()
	}
	pc.ARMDeploymentName = uuid.NewV4().String()
	pc.ServerName = uuid.NewV4().String()
	pc.AdministratorLoginPassword = generate.NewPassword()
	pc.DatabaseName = generate.NewIdentifier()
	return pc, nil
}

func (m *module) deployARMTemplate(
	ctx context.Context, // nolint: unparam
	instanceID string, // nolint: unparam
	serviceID string, // nolint: unparam
	planID string, // nolint: unparam
	provisioningContext service.ProvisioningContext,
	provisioningParameters service.ProvisioningParameters,
) (service.ProvisioningContext, error) {
	pc, ok := provisioningContext.(*mysqlProvisioningContext)
	if !ok {
		return nil, errors.New(
			"error casting provisioningContext as *mysqlProvisioningContext",
		)
	}
	pp, ok := provisioningParameters.(*ProvisioningParameters)
	if !ok {
		return nil, errors.New(
			"error casting provisioningParameters as " +
				"*mysql.ProvisioningParameters",
		)
	}

	catalog, err := m.GetCatalog()
	if err != nil {
		return nil, fmt.Errorf("error retrieving catalog: %s", err)
	}
	service, ok := catalog.GetService(serviceID)
	if !ok {
		return nil, fmt.Errorf(
			`service "%s" not found in the "%s" module catalog`,
			serviceID,
			m.GetName(),
		)
	}
	plan, ok := service.GetPlan(planID)
	if !ok {
		return nil, fmt.Errorf(
			`plan "%s" not found for service "%s"`,
			planID,
			serviceID,
		)
	}
	outputs, err := m.armDeployer.Deploy(
		pc.ARMDeploymentName,
		pc.ResourceGroupName,
		pp.Location,
		armTemplateBytes,
		map[string]interface{}{
			"administratorLoginPassword": pc.AdministratorLoginPassword,
			"serverName":                 pc.ServerName,
			"databaseName":               pc.DatabaseName,
			"skuName":                    plan.GetProperties().Extended["skuName"],
			"skuTier":                    plan.GetProperties().Extended["skuTier"],
			"skuCapacityDTU": plan.GetProperties().
				Extended["skuCapacityDTU"],
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error deploying ARM template: %s", err)
	}

	fullyQualifiedDomainName, ok := outputs["fullyQualifiedDomainName"].(string)
	if !ok {
		return nil, fmt.Errorf(
			"error retrieving fully qualified domain name from deployment: %s",
			err,
		)
	}
	pc.FullyQualifiedDomainName = fullyQualifiedDomainName

	return pc, nil
}
