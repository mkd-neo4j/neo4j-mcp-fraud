package docs

import (
	_ "embed"
)

// QualificationQuestionsPrompt embeds the qualification questions guidance
// This prompt provides behavioral guidance for LLMs on how to think and interact
// with users when processing financial crime investigation queries
//
//go:embed prompts/qualification_questions.md
var QualificationQuestionsPrompt string
