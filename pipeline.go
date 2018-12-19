package main

import (
	"fmt"
	"github.com/concourse/go-concourse/concourse"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataPipeline() *schema.Resource {
	return &schema.Resource{
		Read: dataPipelineRead,

		Schema: map[string]*schema.Schema{
			"pipeline_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"team_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"is_exposed": &schema.Schema{
				Type:     schema.TypeBool,
				Required: false,
				Computed: true,
			},

			"is_paused": &schema.Schema{
				Type:     schema.TypeBool,
				Required: false,
				Computed: true,
			},

			"yaml": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Computed: true,
			},

			"json": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Computed: true,
			},
		},
	}
}

func resourcePipeline() *schema.Resource {
	return &schema.Resource{
		Create: resourcePipelineCreate,
		Read:   resourcePipelineRead,
		Update: resourcePipelineUpdate,
		Delete: resourcePipelineDelete,

		Schema: map[string]*schema.Schema{
			"pipeline_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"team_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"is_exposed": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},

			"is_paused": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},

			"pipeline_config_format": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"pipeline_config": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"json": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"yaml": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

type pipelineHelper struct {
	TeamName     string
	PipelineName string
	IsExposed    bool
	IsPaused     bool
	JSON         string
	YAML         string
}

func pipelineID(teamName string, pipelineName string) string {
	return fmt.Sprintf("%s:%s", teamName, pipelineName)
}

func readPipeline(
	client concourse.Client,
	teamName string,
	pipelineName string,
) (pipelineHelper, error) {

	retVal := pipelineHelper{
		TeamName:     teamName,
		PipelineName: pipelineName,
	}

	team := client.Team(teamName)

	pipeline, pipelineFound, err := team.Pipeline(pipelineName)

	if err != nil {
		return retVal, err
	}

	if !pipelineFound {
		return retVal, fmt.Errorf(
			"Could not find pipeline %s within team %s", pipelineName, teamName,
		)
	}

	_, pipelineCfg, _, pipelineCfgFound, err := team.PipelineConfig(pipelineName)

	if err != nil {
		return retVal, fmt.Errorf(
			"Error looking up pipeline %s within team '%s': %s",
			pipelineName, teamName, err,
		)
	}

	if !pipelineCfgFound {
		return retVal, fmt.Errorf(
			"No pipeline %s found within team %s",
			pipelineName, teamName,
		)
	}

	pipelineCfgJSON, err := JSONToJSON(string(pipelineCfg))
	if err != nil {
		return retVal, fmt.Errorf(
			"Encountered error parsing pipeline %s config within team '%s': %s",
			pipelineName, teamName, err,
		)
	}

	pipelineCfgYAML, err := JSONToYAML(pipelineCfgJSON)

	if err != nil {
		return retVal, fmt.Errorf(
			"Encountered error parsing pipeline %s config within team '%s': %s",
			pipelineName, teamName, err,
		)
	}

	retVal.IsExposed = pipeline.Public
	retVal.IsPaused = pipeline.Paused
	retVal.JSON = pipelineCfgJSON
	retVal.YAML = pipelineCfgYAML

	return retVal, nil
}

func dataPipelineRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*ProviderConfig).Client
	pipelineName := d.Get("pipeline_name").(string)
	teamName := d.Get("team_name").(string)

	pipeline, err := readPipeline(client, teamName, pipelineName)

	if err != nil {
		return fmt.Errorf(
			"Error reading pipeline %s from team '%s': %s",
			pipelineName, teamName, err,
		)
	}

	d.SetId(pipelineID(teamName, pipelineName))
	d.Set("is_exposed", pipeline.IsExposed)
	d.Set("is_paused", pipeline.IsPaused)
	d.Set("json", pipeline.JSON)
	d.Set("yaml", pipeline.YAML)

	return nil
}

func resourcePipelineCreate(d *schema.ResourceData, m interface{}) error {
	return resourcePipelineUpdate(d, m)
}

func resourcePipelineRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*ProviderConfig).Client
	pipelineName := d.Get("pipeline_name").(string)
	teamName := d.Get("team_name").(string)

	pipeline, err := readPipeline(client, teamName, pipelineName)

	if err != nil {
		return fmt.Errorf(
			"Error reading pipeline %s from team '%s': %s",
			pipelineName, teamName, err,
		)
	}

	d.SetId(pipelineID(teamName, pipelineName))
	d.Set("is_exposed", pipeline.IsExposed)
	d.Set("is_paused", pipeline.IsPaused)
	d.Set("json", pipeline.JSON)
	d.Set("yaml", pipeline.YAML)

	return nil
}

func resourcePipelineUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ProviderConfig).Client
	pipelineName := d.Get("pipeline_name").(string)
	teamName := d.Get("team_name").(string)
	d.SetId(pipelineID(teamName, pipelineName))
	team := client.Team(teamName)

	pipelineConfig := d.Get("pipeline_config").(string)
	pipelineConfigFormat := d.Get("pipeline_config_format").(string)

	parsedJSON, err := ParsePipelineConfig(pipelineConfig, pipelineConfigFormat)

	if err != nil {
		return fmt.Errorf("Error parsing pipeline_config: %s", err)
	}

	_, not_created, configWarnings, err := team.CreateOrUpdatePipelineConfig(
		pipelineName, "0", []byte(parsedJSON), false,
	)

	if not_created {
		return fmt.Errorf(
			"Encountered error setting config for pipeline %s in team '%s': %s",
			pipelineName, teamName, err,
		)
	}

	if len(configWarnings) != 0 {
		warnings := ""
		for _, w := range configWarnings {
			warnings += fmt.Sprintf("%s: %s\n", w.Type, w.Message)
		}

		return fmt.Errorf(
			"Encountered pipeline warnings (%s/%s):\n %s",
			pipelineName, teamName, warnings,
		)
	}

	if not_created {
		return fmt.Errorf(
			"Could not create/update pipeline %s in team %s",
			pipelineName, teamName,
		)
	}

	if d.Get("is_exposed").(bool) {
		found, err := team.ExposePipeline(pipelineName)
		if err != nil {
			return fmt.Errorf(
				"Error exposing pipeline %s in team '%s': %s",
				pipelineName, teamName, err,
			)
		}
		if !found {
			return fmt.Errorf(
				"Could not find pipeline %s in team '%s': %s",
				pipelineName, teamName, err,
			)
		}
	} else {
		found, err := team.HidePipeline(pipelineName)
		if err != nil {
			return fmt.Errorf(
				"Error hiding pipeline %s in team '%s': %s",
				pipelineName, teamName, err,
			)
		}
		if !found {
			return fmt.Errorf(
				"Could not find pipeline %s in team '%s': %s",
				pipelineName, teamName, err,
			)
		}
	}

	if d.Get("is_paused").(bool) {
		found, err := team.PausePipeline(pipelineName)
		if err != nil {
			return fmt.Errorf(
				"Error pausing pipeline %s in team '%s': %s",
				pipelineName, teamName, err,
			)
		}
		if !found {
			return fmt.Errorf(
				"Could not find pipeline %s in team '%s': %s",
				pipelineName, teamName, err,
			)
		}
	} else {
		found, err := team.UnpausePipeline(pipelineName)
		if err != nil {
			return fmt.Errorf(
				"Error unpausing pipeline %s in team '%s': %s",
				pipelineName, teamName, err,
			)
		}
		if !found {
			return fmt.Errorf(
				"Could not find pipeline %s in team '%s': %s",
				pipelineName, teamName, err,
			)
		}
	}

	return resourcePipelineRead(d, m)
}

func resourcePipelineDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*ProviderConfig).Client
	pipelineName := d.Get("pipeline_name").(string)
	teamName := d.Get("team_name").(string)
	team := client.Team(teamName)

	deleted, err := team.DeletePipeline(pipelineName)

	if err != nil {
		return fmt.Errorf(
			"Could not delete pipeline %s from team %s: %s",
			pipelineName, teamName, err,
		)
	}

	if !deleted {
		return fmt.Errorf(
			"Could not delete pipeline %s from team %s", pipelineName, teamName,
		)
	}

	d.SetId("")
	return nil
}
