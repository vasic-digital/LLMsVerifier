import { Component } from '@angular/core';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss']
})
export class DashboardComponent {
  title = 'LLM Verifier Dashboard';

  refreshData(): void {
    // In a real implementation, this would refresh data from the API
    console.log('Refreshing dashboard data...');
    // For now, just reload the page
    window.location.reload();
  }
}