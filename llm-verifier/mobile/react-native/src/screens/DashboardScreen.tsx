import React, {useEffect, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  RefreshControl,
  Alert,
} from 'react-native';
import {API_BASE_URL} from '../config';

interface DashboardStats {
  total_models: number;
  verified_models: number;
  pending_verifications: number;
  recent_verifications: number;
  system_health: string;
}

const DashboardScreen = () => {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const loadDashboardStats = async () => {
    try {
      // This would normally fetch from the API
      // For now, we'll use mock data
      const mockStats: DashboardStats = {
        total_models: 25,
        verified_models: 18,
        pending_verifications: 3,
        recent_verifications: 7,
        system_health: 'healthy',
      };

      setStats(mockStats);
    } catch (error) {
      Alert.alert('Error', 'Failed to load dashboard stats');
    }
  };

  const onRefresh = async () => {
    setRefreshing(true);
    await loadDashboardStats();
    setRefreshing(false);
  };

  useEffect(() => {
    loadDashboardStats();
  }, []);

  return (
    <ScrollView
      style={styles.container}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
      }>
      <Text style={styles.title}>LLM Verifier Dashboard</Text>

      {stats && (
        <View style={styles.statsContainer}>
          <View style={styles.statCard}>
            <Text style={styles.statNumber}>{stats.total_models}</Text>
            <Text style={styles.statLabel}>Total Models</Text>
          </View>

          <View style={styles.statCard}>
            <Text style={styles.statNumber}>{stats.verified_models}</Text>
            <Text style={styles.statLabel}>Verified</Text>
          </View>

          <View style={styles.statCard}>
            <Text style={styles.statNumber}>{stats.pending_verifications}</Text>
            <Text style={styles.statLabel}>Pending</Text>
          </View>

          <View style={styles.statCard}>
            <Text style={styles.statNumber}>{stats.recent_verifications}</Text>
            <Text style={styles.statLabel}>Recent</Text>
          </View>
        </View>
      )}

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>System Status</Text>
        <View style={[styles.statusIndicator,
          stats?.system_health === 'healthy' ? styles.statusHealthy :
          stats?.system_health === 'degraded' ? styles.statusDegraded : styles.statusUnhealthy
        ]}>
          <Text style={styles.statusText}>
            {stats?.system_health?.toUpperCase() || 'UNKNOWN'}
          </Text>
        </View>
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Quick Actions</Text>
        <View style={styles.actionButtons}>
          <Text style={styles.actionButton}>Start Verification</Text>
          <Text style={styles.actionButton}>View Reports</Text>
          <Text style={styles.actionButton}>Manage Models</Text>
        </View>
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
    padding: 16,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    marginBottom: 20,
    color: '#333',
  },
  statsContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'space-between',
    marginBottom: 20,
  },
  statCard: {
    backgroundColor: 'white',
    borderRadius: 8,
    padding: 16,
    width: '48%',
    marginBottom: 16,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  statNumber: {
    fontSize: 32,
    fontWeight: 'bold',
    color: '#007AFF',
    textAlign: 'center',
  },
  statLabel: {
    fontSize: 14,
    color: '#666',
    textAlign: 'center',
    marginTop: 8,
  },
  section: {
    backgroundColor: 'white',
    borderRadius: 8,
    padding: 16,
    marginBottom: 16,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: 'bold',
    marginBottom: 12,
    color: '#333',
  },
  statusIndicator: {
    paddingVertical: 8,
    paddingHorizontal: 16,
    borderRadius: 4,
    alignSelf: 'flex-start',
  },
  statusHealthy: {
    backgroundColor: '#d4edda',
  },
  statusDegraded: {
    backgroundColor: '#fff3cd',
  },
  statusUnhealthy: {
    backgroundColor: '#f8d7da',
  },
  statusText: {
    fontSize: 14,
    fontWeight: 'bold',
    color: '#333',
  },
  actionButtons: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  actionButton: {
    backgroundColor: '#007AFF',
    color: 'white',
    paddingVertical: 12,
    paddingHorizontal: 16,
    borderRadius: 6,
    fontSize: 14,
    fontWeight: 'bold',
    textAlign: 'center',
    flex: 1,
    marginHorizontal: 4,
  },
});

export default DashboardScreen;