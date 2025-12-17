import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/verification_provider.dart';
import '../widgets/model_card.dart';

class ModelsScreen extends StatefulWidget {
  const ModelsScreen({super.key});

  @override
  State<ModelsScreen> createState() => _ModelsScreenState();
}

class _ModelsScreenState extends State<ModelsScreen> {
  final TextEditingController _searchController = TextEditingController();
  String _selectedProvider = 'All';
  String _selectedCategory = 'All';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<VerificationProvider>().loadModels();
    });
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Models'),
        actions: [
          IconButton(
            icon: const Icon(Icons.filter_list),
            onPressed: _showFilterDialog,
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                hintText: 'Search models...',
                prefixIcon: const Icon(Icons.search),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
                filled: true,
                fillColor: Theme.of(context).colorScheme.surface,
              ),
              onChanged: (value) {
                setState(() {});
              },
            ),
          ),
          Expanded(
            child: Consumer<VerificationProvider>(
              builder: (context, provider, _) {
                if (provider.isLoading) {
                  return const Center(child: CircularProgressIndicator());
                }

                var filteredModels = provider.allModels.where((model) {
                  final matchesSearch = _searchController.text.isEmpty ||
                      model.name.toLowerCase().contains(_searchController.text.toLowerCase()) ||
                      model.provider.toLowerCase().contains(_searchController.text.toLowerCase());

                  final matchesProvider = _selectedProvider == 'All' ||
                      model.provider == _selectedProvider;

                  final matchesCategory = _selectedCategory == 'All' ||
                      model.category == _selectedCategory;

                  return matchesSearch && matchesProvider && matchesCategory;
                }).toList();

                if (filteredModels.isEmpty) {
                  return const Center(
                    child: Text('No models found'),
                  );
                }

                return RefreshIndicator(
                  onRefresh: () => provider.loadModels(),
                  child: ListView.builder(
                    padding: const EdgeInsets.all(16),
                    itemCount: filteredModels.length,
                    itemBuilder: (context, index) {
                      return ModelCard(model: filteredModels[index]);
                    },
                  ),
                );
              },
            ),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () => Navigator.pushNamed(context, '/verification'),
        child: const Icon(Icons.play_arrow),
      ),
    );
  }

  void _showFilterDialog() {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Filter Models'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            DropdownButtonFormField<String>(
              value: _selectedProvider,
              decoration: const InputDecoration(labelText: 'Provider'),
              items: ['All', 'OpenAI', 'Anthropic', 'DeepSeek', 'Google']
                  .map((provider) => DropdownMenuItem(
                        value: provider,
                        child: Text(provider),
                      ))
                  .toList(),
              onChanged: (value) {
                setState(() => _selectedProvider = value!);
              },
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              value: _selectedCategory,
              decoration: const InputDecoration(labelText: 'Category'),
              items: ['All', 'coding', 'reasoning', 'chat', 'multimodal', 'generative']
                  .map((category) => DropdownMenuItem(
                        value: category,
                        child: Text(category),
                      ))
                  .toList(),
              onChanged: (value) {
                setState(() => _selectedCategory = value!);
              },
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Close'),
          ),
        ],
      ),
    );
  }
}