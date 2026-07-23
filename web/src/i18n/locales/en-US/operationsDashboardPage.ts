const operationsDashboardPage = {
  description: 'View the Grafana operations dashboard configured by an administrator.',
  configure: 'Configure dashboard',
  emptyTitle: 'No operations dashboard configured',
  emptyDescription: 'Add a Grafana dashboard or panel iframe URL in Global Settings to show it here.',
  invalidTitle: 'Invalid operations dashboard URL',
  invalidDescription: 'Enter a Grafana iframe URL that starts with http or https in Global Settings.',
  loadFailedTitle: 'Operations dashboard failed to load',
  loadFailedDescription: 'Confirm the current account has platform administrator permissions, or try again later.',
  iframeTimeoutTitle: 'The operations dashboard has not finished loading',
  iframeTimeoutDescription: 'Cross-origin embeds cannot expose reliable error details. Retry, open it in a new window, or check the Grafana embedding and authentication settings.',
  openInNewWindow: 'Open in new window',
}

export default operationsDashboardPage
