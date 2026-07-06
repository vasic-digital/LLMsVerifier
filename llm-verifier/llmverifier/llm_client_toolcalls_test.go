package llmverifier

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// toolCallsResponseFixture is a realistic OpenAI-style chat/completions
// response whose assistant message carries a `tool_calls` array (the model
// invoking a function). Content is null — the shape OpenAI/Groq/Together/etc.
// return when a model responds ONLY with tool calls.
const toolCallsResponseFixture = `{
  "id": "chatcmpl-toolcalls-red",
  "object": "chat.completion",
  "created": 1699999999,
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": null,
        "tool_calls": [
          {
            "id": "call_weather_001",
            "type": "function",
            "function": {
              "name": "get_current_weather",
              "arguments": "{\"location\":\"New York\"}"
            }
          },
          {
            "id": "call_stock_002",
            "type": "function",
            "function": {
              "name": "get_stock_price",
              "arguments": "{\"symbol\":\"AAPL\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ],
  "usage": {"prompt_tokens": 20, "completion_tokens": 12, "total_tokens": 32}
}`

// TestMessageToolCallsRoundTrip is the §11.4.115 RED-baseline-on-the-broken-
// artifact polarity test for change C1 (CONST-039/040: tool-calling
// verification). It reproduces the defect that a response's `tool_calls`
// array is DROPPED when unmarshalled into ChatCompletionResponse, because
// the Message struct carried only Role+Content (llm_client.go:92-95).
//
// The oracle is a JSON round-trip (parse → re-marshal the parsed Message):
// if the field is carried, the re-marshalled JSON still contains the tool
// call ids + function names; if the field is dropped, they vanish. This
// oracle references no ToolCalls field, so the SAME source compiles on both
// the pre-fix (broken) and post-fix (fixed) artifacts.
//
// Polarity switch (§11.4.115):
//
//	RED_MODE=1            — reproduce the defect: assert tool_calls are
//	    ABSENT after round-trip. PASSes on the BROKEN artifact (proof the
//	    defect is real), FAILs on the FIXED artifact. The RED capture on the
//	    pre-fix artifact was run with this mode.
//	RED_MODE=0 (default)  — standing GREEN regression guard: assert
//	    tool_calls SURVIVE the round-trip and Message.ToolCalls is populated.
//	    FAILs on BROKEN, PASSes on FIXED. Default so `go test ./...` runs the
//	    standing guard GREEN on the fixed tree per §11.4.135 (guard runs on
//	    every build); RED_MODE=1 remains the explicit reproduction opt-in.
func TestMessageToolCallsRoundTrip(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}

	var resp ChatCompletionResponse
	if err := json.Unmarshal([]byte(toolCallsResponseFixture), &resp); err != nil {
		t.Fatalf("failed to unmarshal tool_calls fixture: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}

	// Re-marshal the PARSED assistant message. What survives here is exactly
	// what the parsed struct retained — the drop-detection oracle.
	roundTripped, err := json.Marshal(resp.Choices[0].Message)
	if err != nil {
		t.Fatalf("failed to re-marshal parsed message: %v", err)
	}
	got := string(roundTripped)

	carriesWeather := strings.Contains(got, "get_current_weather")
	carriesStock := strings.Contains(got, "get_stock_price")
	carriesCallID := strings.Contains(got, "call_weather_001")
	toolCallsSurvived := carriesWeather && carriesStock && carriesCallID

	switch redMode {
	case "1":
		// Reproduce-and-assert-defect-present on the broken artifact.
		if toolCallsSurvived {
			t.Fatalf("RED_MODE=1: expected tool_calls to be DROPPED (defect present) "+
				"but they survived the round-trip: %s", got)
		}
		t.Logf("RED_MODE=1 PASS: defect reproduced — tool_calls dropped by parse. "+
			"Round-tripped message = %s", got)
	case "0":
		// Standing GREEN regression guard on the fixed artifact.
		if !toolCallsSurvived {
			t.Fatalf("RED_MODE=0: expected tool_calls to SURVIVE the round-trip "+
				"(fixed) but they were dropped: %s", got)
		}
		// Direct field assertion — the parsed Message.ToolCalls must be
		// populated with both invoked functions, in order.
		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) != 2 {
			t.Fatalf("RED_MODE=0: expected Message.ToolCalls length 2, got %d", len(msg.ToolCalls))
		}
		if msg.ToolCalls[0].ID != "call_weather_001" ||
			msg.ToolCalls[0].Type != "function" ||
			msg.ToolCalls[0].Function.Name != "get_current_weather" ||
			!strings.Contains(msg.ToolCalls[0].Function.Arguments, "New York") {
			t.Fatalf("RED_MODE=0: tool call[0] mis-parsed: %+v", msg.ToolCalls[0])
		}
		if msg.ToolCalls[1].Function.Name != "get_stock_price" ||
			!strings.Contains(msg.ToolCalls[1].Function.Arguments, "AAPL") {
			t.Fatalf("RED_MODE=0: tool call[1] mis-parsed: %+v", msg.ToolCalls[1])
		}
		t.Logf("RED_MODE=0 PASS: Message.ToolCalls populated with %d real tool calls "+
			"(%s, %s). Round-tripped message = %s", len(msg.ToolCalls),
			msg.ToolCalls[0].Function.Name, msg.ToolCalls[1].Function.Name, got)
	default:
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}
}
