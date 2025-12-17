import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/verification_provider.dart';

class VerificationSummary extends StatelessWidget {
  const VerificationSummary({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<VerificationProvider>(
      builder: (context, provider, _) {
        final stats = provider.dashboardStats;
        final averageScore = provider.getAverageScore();

        return Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Verification Summary',
                  style: Theme.of(context).textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: _buildStatCard(
                        context,
                        'Total Models',
                        stats?.totalModels.toString() ?? provider.allModels.length.toString(),
                        Icons.list,
                        Colors.blue,
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _buildStatCard(
                        context,
                        'Verified',
                        stats?.verifiedModels.toString() ?? '0',
                        Icons.check_circle,
                        Colors.green,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: _buildStatCard(
                        context,
                        'Average Score',
                        averageScore > 0 ? '${averageScore.toStringAsFixed(1)}' : 'N/A',
                        Icons.score,
                        Colors.orange,
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _buildStatCard(
                        context,
                        'Recent Runs',
                        stats?.recentVerifications.toString() ?? '0',
                        Icons.history,
                        Colors.purple,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                LinearProgressIndicator(
                  value: averageScore > 0 ? averageScore / 100 : 0,
                  backgroundColor: Theme.of(context).colorScheme.surfaceContainerHighest,
                  valueColor: AlwaysStoppedAnimation<Color>(
                    averageScore >= 85 ? Colors.green :
                    averageScore >= 70 ? Colors.blue :
                    averageScore >= 50 ? Colors.orange : Colors.red,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Overall Model Quality',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildStatCard(BuildContext context, String title, String value, IconData icon, Color color) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        children: [
          Icon(icon, color: color, size: 24),
          const SizedBox(height: 8),
          Text(
            value,
            style: Theme.of(context).textTheme.headlineSmall?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            title,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }
}