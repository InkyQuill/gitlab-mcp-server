package gitlab

import (
	"testing"
)

func BenchmarkParseLabelString_Single(b *testing.B) {
	labels := "bug"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseLabelString(labels)
	}
}

func BenchmarkParseLabelString_Multiple(b *testing.B) {
	labels := "bug,enhancement,priority::high"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseLabelString(labels)
	}
}

func BenchmarkParseLabelString_WithSpaces(b *testing.B) {
	labels := "bug, enhancement, priority::high"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseLabelString(labels)
	}
}

func BenchmarkParseLabelString_Empty(b *testing.B) {
	labels := ""
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseLabelString(labels)
	}
}

func BenchmarkParseAssigneeIDsString_Single(b *testing.B) {
	ids := "1"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseAssigneeIDsString(ids)
	}
}

func BenchmarkParseAssigneeIDsString_Multiple(b *testing.B) {
	ids := "1,2,3,4,5"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseAssigneeIDsString(ids)
	}
}

func BenchmarkParseAssigneeIDsString_WithSpaces(b *testing.B) {
	ids := "1, 2, 3, 4, 5"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseAssigneeIDsString(ids)
	}
}

func BenchmarkParseAssigneeIDsString_Empty(b *testing.B) {
	ids := ""
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseAssigneeIDsString(ids)
	}
}

func BenchmarkValidateAndConvertMilestoneID_Valid(b *testing.B) {
	id := 123.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateAndConvertMilestoneID(id)
	}
}

func BenchmarkValidateAndConvertMilestoneID_Zero(b *testing.B) {
	id := 0.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateAndConvertMilestoneID(id)
	}
}

func BenchmarkParseDueDate_Valid(b *testing.B) {
	date := "2024-12-31"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseDueDate(date)
	}
}

func BenchmarkParseDueDate_Empty(b *testing.B) {
	date := ""
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseDueDate(date)
	}
}

func BenchmarkRequiredFloatToIntParam_Valid(b *testing.B) {
	val := 42.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = RequiredFloatToIntParam(val, "testParam")
	}
}

func BenchmarkOptionalFloatToIntParam_Valid(b *testing.B) {
	val := 42.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = OptionalFloatToIntParam(val, "testParam")
	}
}

func BenchmarkOptionalFloatToIntParam_Zero(b *testing.B) {
	val := 0.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = OptionalFloatToIntParam(val, "testParam")
	}
}
