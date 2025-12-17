import 'package:flutter/material.dart';
import '../models/models.dart';

class ModelCard extends StatelessWidget {
  final Model model;

  const ModelCard({super.key, required this.model});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        model.name,
                        style: Theme.of(context).textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        model.provider,
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: Theme.of(context).colorScheme.primary,
                        ),
                      ),
                    ],
                  ),
                ),
                _buildScoreBadge(),
              ],
            ),
            const SizedBox(height: 12),
            Text(
              model.description,
              style: Theme.of(context).textTheme.bodyMedium,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 4,
              children: [
                _buildCategoryChip(),
                if (model.supportsVision) _buildCapabilityChip('Vision'),
                if (model.supportsAudio) _buildCapabilityChip('Audio'),
                if (model.supportsVideo) _buildCapabilityChip('Video'),
                if (model.supportsReasoning) _buildCapabilityChip('Reasoning'),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildScoreBadge() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: _getScoreColor(),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        '${model.overallScore.toStringAsFixed(1)}',
        style: const TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.bold,
          fontSize: 14,
        ),
      ),
    );
  }

  Color _getScoreColor() {
    if (model.overallScore >= 85) return Colors.green;
    if (model.overallScore >= 70) return Colors.blue;
    if (model.overallScore >= 50) return Colors.orange;
    return Colors.red;
  }

  Widget _buildCategoryChip() {
    return Chip(
      label: Text(
        model.category.toUpperCase(),
        style: const TextStyle(fontSize: 12),
      ),
      backgroundColor: Theme.of(context).colorScheme.secondaryContainer,
      side: BorderSide.none,
    );
  }

  Widget _buildCapabilityChip(String capability) {
    return Chip(
      label: Text(
        capability,
        style: const TextStyle(fontSize: 12),
      ),
      backgroundColor: Theme.of(context).colorScheme.surfaceContainerHighest,
      side: BorderSide.none,
    );
  }
}