package utils

import (
	"encoding/json"
	"fmt"

	"github.com/jfrog/jfrog-cli-core/v2/jobsummaries"
	"github.com/jfrog/jfrog-cli-security/formats"
)

const (
	Build   SecuritySummarySection = "Builds"
	Binary  SecuritySummarySection = "Binaries"
	Modules SecuritySummarySection = "Modules"
)

type SecuritySummarySection string

type ScanCommandSummaryResult struct {
	Section SecuritySummarySection
	Results formats.SummaryResults
}

type SecurityCommandsSummary struct {
	BuildScanCommands []formats.SummaryResults `json:"buildScanCommands"`
	ScanCommands      []formats.SummaryResults `json:"scanCommands"`
	AuditCommands     []formats.SummaryResults `json:"auditCommands"`
}

func (scs *SecurityCommandsSummary) CreateSummaryMarkdown(content any, section jobsummaries.MarkdownSection) (err error) {

	previousObjects, err := jobsummaries.LoadFile(jobsummaries.GetSectionFileName(section))
	if err != nil {
		return fmt.Errorf("failed to load previous objects: %w", err)
	}

	dataAsBytes, err := scs.appendResultObject(content, previousObjects)
	if err != nil {
		return fmt.Errorf("failed to parase markdown section objects: %w", err)
	}

	if err = jobsummaries.WriteFile(dataAsBytes, jobsummaries.GetSectionFileName(section)); err != nil {
		return fmt.Errorf("failed to write aggregated data to file: %w", err)
	}

	markdown, err := scs.renderContentToMarkdown(dataAsBytes)
	if err != nil {
		return fmt.Errorf("failed to render markdown :%w", err)
	}

	if err = jobsummaries.WriteMarkdownToFileSystem(markdown, scs.GetSectionTitle(), section); err != nil {
		return fmt.Errorf("failed to save markdown to file system")
	}
	return err
}

// Manage the job summary for security commands
func SecurityCommandsJobSummary() (js *jobsummaries.JobSummary, err error) {
	return jobsummaries.NewJobSummaryImpl(&SecurityCommandsSummary{
		BuildScanCommands: []formats.SummaryResults{},
		ScanCommands:      []formats.SummaryResults{},
		AuditCommands:     []formats.SummaryResults{},
	})
}

// Record the security command output
func RecordSecurityCommandOutput(content ScanCommandSummaryResult) (err error) {
	manager, err := SecurityCommandsJobSummary()
	if err != nil || manager == nil {
		return
	}
	return manager.CreateSummaryMarkdown(content, jobsummaries.SecuritySection)
}

func (scs *SecurityCommandsSummary) GetSectionTitle() string {
	return "🛡️ Security scans preformed by this job"
}

func (scs *SecurityCommandsSummary) appendResultObject(output interface{}, previousObjects []byte) (result []byte, err error) {
	// Unmarshal the aggregated data
	if len(previousObjects) > 0 {
		if err = json.Unmarshal(previousObjects, &scs); err != nil {
			return
		}
	}
	// Append the new data
	data, ok := output.(ScanCommandSummaryResult)
	if !ok {
		err = fmt.Errorf("failed to cast output to ScanCommandSummaryResult")
		return
	}
	switch data.Section {
	case Build:
		scs.BuildScanCommands = append(scs.BuildScanCommands, data.Results)
	case Binary:
		scs.ScanCommands = append(scs.ScanCommands, data.Results)
	case Modules:
		scs.AuditCommands = append(scs.AuditCommands, data.Results)
	}
	return json.Marshal(scs)
}

func (scs *SecurityCommandsSummary) renderContentToMarkdown(content []byte) (markdown string, err error) {
	// Unmarshal the data into an array of build info objects
	if err = json.Unmarshal(content, &scs); err != nil {
		return "", fmt.Errorf("failed while creating security markdown: %w", err)
	}
	return ConvertSummaryToString(*scs)
}

func (scs *SecurityCommandsSummary) GetOrderedSectionsWithContent() (sections []SecuritySummarySection) {
	if len(scs.BuildScanCommands) > 0 {
		sections = append(sections, Build)
	}
	if len(scs.ScanCommands) > 0 {
		sections = append(sections, Binary)
	}
	if len(scs.AuditCommands) > 0 {
		sections = append(sections, Modules)
	}
	return

}

func (scs *SecurityCommandsSummary) GetSectionSummaries(section SecuritySummarySection) (summaries []formats.SummaryResults) {
	switch section {
	case Build:
		summaries = scs.BuildScanCommands
	case Binary:
		summaries = scs.ScanCommands
	case Modules:
		summaries = scs.AuditCommands
	}
	return
}

func ConvertSummaryToString(results SecurityCommandsSummary) (summary string, err error) {
	sectionsWithContent := results.GetOrderedSectionsWithContent()
	addSectionTitle := len(sectionsWithContent) > 1
	var sectionSummary string
	for i, section := range sectionsWithContent {
		if sectionSummary, err = convertScanSectionToString(results.GetSectionSummaries(section)...); err != nil {
			return
		}
		if addSectionTitle {
			if i > 0 {
				summary += "\n"
			}
			summary += fmt.Sprintf("#### %s\n", section)
		}
		summary += sectionSummary
	}
	return
}

func convertScanSectionToString(results ...formats.SummaryResults) (summary string, err error) {
	if len(results) == 0 {
		return
	}
	content, err := GetSummaryString(results...)
	if err != nil {
		return
	}
	summary = fmt.Sprintf("```\n%s\n```", content)
	return
}
