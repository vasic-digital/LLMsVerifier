import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/verification_provider.dart';
import '../models/models.dart';

class VerificationScreen extends StatefulWidget {
  const VerificationScreen({super.key});

  @override
  State<VerificationScreen> createState() => _VerificationScreenState();
}

class _VerificationScreenState extends State<VerificationScreen> {
  Model? _selectedModel;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Start Verification'),
      ),
      body: Consumer<VerificationProvider>(
        builder: (context, provider, _) {
          final models = provider.allModels.where((m) => m.overallScore > 0).toList();

          return Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'Select a model to verify',
                  style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 16),
                DropdownButtonFormField<Model>(
                  value: _selectedModel,
                  decoration: const InputDecoration(
                    labelText: 'Model',
                    border: OutlineInputBorder(),
                  ),
                  items: models.map((model) {
                    return DropdownMenuItem<Model>(
                      value: model,
                      child: Text('${model.name} (${model.provider})'),
                    );
                  }).toList(),
                  onChanged: (model) {
                    setState(() => _selectedModel = model);
                  },
                ),
                const SizedBox(height: 24),
                if (_selectedModel != null) ...[
                  Card(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'Model Details',
                            style: Theme.of(context).textTheme.titleMedium,
                          ),
                          const SizedBox(height: 8),
                          Text('Name: ${_selectedModel!.name}'),
                          Text('Provider: ${_selectedModel!.provider}'),
                          Text('Current Score: ${_selectedModel!.overallScore.toStringAsFixed(1)}'),
                          Text('Category: ${_selectedModel!.category}'),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 24),
                  FilledButton.icon(
                    onPressed: provider.isLoading ? null : _startVerification,
                    icon: const Icon(Icons.play_arrow),
                    label: provider.isLoading
                        ? const Text('Starting Verification...')
                        : const Text('Start Verification'),
                    style: FilledButton.styleFrom(
                      minimumSize: const Size(double.infinity, 48),
                    ),
                  ),
                ],
              ],
            ),
          );
        },
      ),
    );
  }

  Future<void> _startVerification() async {
    if (_selectedModel == null) return;

    final provider = context.read<VerificationProvider>();

    try {
      final result = await provider.startVerification(_selectedModel!.id);

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Verification started successfully!'),
            backgroundColor: Colors.green,
          ),
        );

        // Navigate back to dashboard
        Navigator.pop(context);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Failed to start verification: $e'),
            backgroundColor: Colors.red,
          ),
        );
      }
    }
  }
}