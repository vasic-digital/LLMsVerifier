import React, {useEffect, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  TouchableOpacity,
  RefreshControl,
} from 'react-native';

interface Model {
  id: number;
  model_id: string;
  name: string;
  provider_name: string;
  verification_status: string;
  overall_score: number;
}

const ModelsScreen = () => {
  const [models, setModels] = useState<Model[]>([]);
  const [refreshing, setRefreshing] = useState(false);

  const loadModels = async () => {
    // Mock data for demonstration
    const mockModels: Model[] = [
      {
        id: 1,
        model_id: 'gpt-4-turbo',
        name: 'GPT-4 Turbo',
        provider_name: 'OpenAI',
        verification_status: 'verified',
        overall_score: 92.5,
      },
      {
        id: 2,
        model_id: 'claude-3-sonnet',
        name: 'Claude 3 Sonnet',
        provider_name: 'Anthropic',
        verification_status: 'verified',
        overall_score: 89.7,
      },
      {
        id: 3,
        model_id: 'gemini-pro',
        name: 'Gemini Pro',
        provider_name: 'Google',
        verification_status: 'pending',
        overall_score: 0,
      },
    ];

    setModels(mockModels);
  };

  const onRefresh = async () => {
    setRefreshing(true);
    await loadModels();
    setRefreshing(false);
  };

  useEffect(() => {
    loadModels();
  }, []);

  const renderModel = ({item}: {item: Model}) => (
    <TouchableOpacity style={styles.modelCard}>
      <View style={styles.modelHeader}>
        <Text style={styles.modelName}>{item.name}</Text>
        <View style={[
          styles.statusBadge,
          item.verification_status === 'verified' ? styles.statusVerified :
          item.verification_status === 'failed' ? styles.statusFailed : styles.statusPending
        ]}>
          <Text style={styles.statusText}>{item.verification_status}</Text>
        </View>
      </View>

      <Text style={styles.providerName}>{item.provider_name}</Text>

      {item.verification_status === 'verified' && (
        <View style={styles.scoreContainer}>
          <Text style={styles.scoreLabel}>Score:</Text>
          <Text style={styles.scoreValue}>{item.overall_score.toFixed(1)}%</Text>
        </View>
      )}
    </TouchableOpacity>
  );

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Models</Text>

      <FlatList
        data={models}
        renderItem={renderModel}
        keyExtractor={(item) => item.id.toString()}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
        contentContainerStyle={styles.listContainer}
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    padding: 16,
    color: '#333',
  },
  listContainer: {
    padding: 16,
  },
  modelCard: {
    backgroundColor: 'white',
    borderRadius: 8,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  modelHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  modelName: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#333',
    flex: 1,
  },
  statusBadge: {
    paddingVertical: 4,
    paddingHorizontal: 8,
    borderRadius: 4,
  },
  statusVerified: {
    backgroundColor: '#d4edda',
  },
  statusFailed: {
    backgroundColor: '#f8d7da',
  },
  statusPending: {
    backgroundColor: '#fff3cd',
  },
  statusText: {
    fontSize: 12,
    fontWeight: 'bold',
    textTransform: 'uppercase',
  },
  providerName: {
    fontSize: 14,
    color: '#666',
    marginBottom: 8,
  },
  scoreContainer: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  scoreLabel: {
    fontSize: 14,
    color: '#666',
    marginRight: 8,
  },
  scoreValue: {
    fontSize: 16,
    fontWeight: 'bold',
    color: '#007AFF',
  },
});

export default ModelsScreen;