import 'package:flutter/foundation.dart';

import '../api/client.dart';
import '../services/token_storage.dart';

class AccountModel extends ChangeNotifier {
  Map<String, dynamic>? _profile;
  List<Map<String, dynamic>> _bankAccounts = [];
  List<Map<String, dynamic>> _transactions = [];

  Map<String, dynamic>? get profile => _profile;
  List<Map<String, dynamic>> get bankAccounts => _bankAccounts;
  List<Map<String, dynamic>> get transactions => _transactions;

  Future<void> login(String email, String password) async {
    final data = await ApiClient.instance
        .post('/auth/login', {'email': email, 'password': password});
    final token = data['access_token'] as String;
    ApiClient.instance.setToken(token);
    await TokenStorage.instance.saveToken(token);
    await load();
  }

  Future<void> load() async {
    _profile = await ApiClient.instance.get('/me');
    _bankAccounts = List<Map<String, dynamic>>.from(
        await ApiClient.instance.get('/bank-accounts'));
    _transactions = List<Map<String, dynamic>>.from(
        await ApiClient.instance.get('/transactions', {'limit': '50'}));
    notifyListeners();
  }

  Future<bool> restore() async {
    final token = await TokenStorage.instance.readToken();
    if (token == null) return false;
    ApiClient.instance.setToken(token);
    try {
      await load();
      return true;
    } on SessionExpired {
      await TokenStorage.instance.clear();
      return false;
    }
  }

  Future<void> logout() async {
    await ApiClient.instance.post('/auth/logout', {});
    ApiClient.instance.setToken(null);
    await TokenStorage.instance.clear();
  }
}
