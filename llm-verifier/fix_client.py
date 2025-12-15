import re

with open('client/client.go', 'r') as f:
    content = f.read()

# Fix GetPricing
content = re.sub(
    r'// GetPricing retrieves pricing information\nfunc \(c \*Client\) GetPricing\(\) \(\[\]map\[string\]interface\{\}, error\) \{\n.*?var pricing \[\]map\[string\]interface\{\}',
    '''// GetPricing retrieves pricing information
func (c *Client) GetPricing() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/pricing", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Pricing []map[string]interface{} \`json:"pricing"\`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode pricing response: %w", err)
	}

	return response.Pricing, nil''',
    content,
    flags=re.DOTALL
)

# Fix GetLimits
content = re.sub(
    r'// GetLimits retrieves rate limit information\nfunc \(c \*Client\) GetLimits\(\) \(\[\]map\[string\]interface\{\}, error\) \{\n.*?var limits \[\]map\[string\]interface\{\}',
    '''// GetLimits retrieves rate limit information
func (c *Client) GetLimits() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/limits", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Limits []map[string]interface{} \`json:"limits"\`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode limits response: %w", err)
	}

	return response.Limits, nil''',
    content,
    flags=re.DOTALL
)

# Fix GetIssues
content = re.sub(
    r'// GetIssues retrieves issue reports\nfunc \(c \*Client\) GetIssues\(\) \(\[\]map\[string\]interface\{\}, error\) \{\n.*?var issues \[\]map\[string\]interface\{\}',
    '''// GetIssues retrieves issue reports
func (c *Client) GetIssues() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/issues", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Issues []map[string]interface{} \`json:"issues"\`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode issues response: %w", err)
	}

	return response.Issues, nil''',
    content,
    flags=re.DOTALL
)

# Fix GetEvents
content = re.sub(
    r'// GetEvents retrieves system events\nfunc \(c \*Client\) GetEvents\(\) \(\[\]map\[string\]interface\{\}, error\) \{\n.*?var events \[\]map\[string\]interface\{\}',
    '''// GetEvents retrieves system events
func (c *Client) GetEvents() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/events", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Events []map[string]interface{} \`json:"events"\`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode events response: %w", err)
	}

	return response.Events, nil''',
    content,
    flags=re.DOTALL
)

# Fix GetSchedules
content = re.sub(
    r'// GetSchedules retrieves verification schedules\nfunc \(c \*Client\) GetSchedules\(\) \(\[\]map\[string\]interface\{\}, error\) \{\n.*?var schedules \[\]map\[string\]interface\{\}',
    '''// GetSchedules retrieves verification schedules
func (c *Client) GetSchedules() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/schedules", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Schedules []map[string]interface{} \`json:"schedules"\`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode schedules response: %w", err)
	}

	return response.Schedules, nil''',
    content,
    flags=re.DOTALL
)

# Fix GetConfigExports
content = re.sub(
    r'// GetConfigExports retrieves configuration exports\nfunc \(c \*Client\) GetConfigExports\(\) \(\[\]map\[string\]interface\{\}, error\) \{\n.*?var exports \[\]map\[string\]interface\{\}',
    '''// GetConfigExports retrieves configuration exports
func (c *Client) GetConfigExports() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/exports", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Exports []map[string]interface{} \`json:"exports"\`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode config exports response: %w", err)
	}

	return response.Exports, nil''',
    content,
    flags=re.DOTALL
)

# Fix GetLogs
content = re.sub(
    r'// GetLogs retrieves system logs\nfunc \(c \*Client\) GetLogs\(\) \(\[\]map\[string\]interface\{\}, error\) \{\n.*?var logs \[\]map\[string\]interface\{\}',
    '''// GetLogs retrieves system logs
func (c *Client) GetLogs() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/logs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Logs []map[string]interface{} \`json:"logs"\`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode logs response: %w", err)
	}

	return response.Logs, nil''',
    content,
    flags=re.DOTALL
)

with open('client/client.go', 'w') as f:
    f.write(content)

print("Fixed all client methods")
