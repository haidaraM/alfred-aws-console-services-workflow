package searchers

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	aw "github.com/deanishe/awgo"
	"github.com/rkoval/alfred-aws-console-services-workflow/awsworkflow"
	"github.com/rkoval/alfred-aws-console-services-workflow/caching"
	"github.com/rkoval/alfred-aws-console-services-workflow/searchers/searchutil"
	"github.com/rkoval/alfred-aws-console-services-workflow/util"
)

type CloudFormationStackSearcher struct{}

func (s CloudFormationStackSearcher) Search(wf *aw.Workflow, searchArgs searchutil.SearchArgs) error {
	cacheName := util.GetCurrentFilename()
	entities := caching.LoadCloudformationStackArrayFromCache(wf, searchArgs, cacheName, s.fetch)
	for _, entity := range entities {
		s.addToWorkflow(wf, searchArgs, entity)
	}
	return nil
}

func (s CloudFormationStackSearcher) fetch(cfg aws.Config) ([]types.Stack, error) {
	svc := cloudformation.NewFromConfig(cfg)

	pageToken := ""
	var entities []types.Stack
	for {
		params := &cloudformation.DescribeStacksInput{}
		if pageToken != "" {
			params.NextToken = aws.String(pageToken)
		}
		resp, err := svc.DescribeStacks(context.TODO(), params)
		if err != nil {
			return nil, err
		}

		entities = append(entities, resp.Stacks...)

		if resp.NextToken != nil {
			pageToken = *resp.NextToken
		} else {
			break
		}
	}

	return entities, nil
}

func (s CloudFormationStackSearcher) addToWorkflow(wf *aw.Workflow, searchArgs searchutil.SearchArgs, entity types.Stack) {
	title := *entity.StackName
	tagName := util.GetCloudFormationTagValue(entity.Tags, "Name")
	// if stack was generated by ElasticBeanstalk and has the generated name, append name tag
	if strings.HasPrefix(title, "awseb-") && tagName != "" {
		title += fmt.Sprintf(" (%s)", tagName)
	}
	subtitle := *entity.Description

	path := fmt.Sprintf("/cloudformation/home#/stacks/stackinfo?stackId=%s", *entity.StackId)
	item := util.NewURLItem(wf, title).
		Subtitle(subtitle).
		Arg(util.ConstructAWSConsoleUrl(path, searchArgs.GetRegion())).
		Icon(awsworkflow.GetImageIcon("cloudwatch"))

	// if stack was generated by ElasticBeanstalk and has the generated name, use the name tag instead
	if strings.HasPrefix(title, "awseb-") && !strings.HasPrefix(searchArgs.Query, "awseb-") && tagName != "" {
		item.Match(tagName)
	} else {
		item.Match(title)
	}
}
