import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class SessionExpired implements Exception {}

class ApiClient {
  ApiClient._();
  static final instance = ApiClient._();

  final _host = 'api.paywallet.example';
  String? _token;

  void setToken(String? token) => _token = token;

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Future<dynamic> get(String path, [Map<String, String>? query]) async {
    final uri = Uri.https(_host, '/v1$path', query);
    if (kDebugMode) debugPrint('GET $path');
    final res = await http.get(uri, headers: _headers);
    if (res.statusCode == 401) throw SessionExpired();
    final body = jsonDecode(res.body);
    if (res.statusCode >= 200 && res.statusCode < 300) return body['data'];
    throw body['message'] ?? 'Có lỗi xảy ra';
  }

  Future<dynamic> post(String path, Map<String, dynamic> data) async {
    final uri = Uri.https(_host, '/v1$path');
    print('POST $path headers=$_headers body=${jsonEncode(data)}');
    final res = await http.post(uri, headers: _headers, body: jsonEncode(data));
    final body = jsonDecode(res.body);
    if (res.statusCode >= 200 && res.statusCode < 300) return body['data'];
    throw body['message'] ?? 'Có lỗi xảy ra';
  }
}
