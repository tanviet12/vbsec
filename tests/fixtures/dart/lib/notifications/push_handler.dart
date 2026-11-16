import 'dart:async';

import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

class PushHandler {
  PushHandler._();
  static final instance = PushHandler._();

  static final _orderId = RegExp(
      r'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$');

  StreamSubscription<RemoteMessage>? _opened;

  Future<void> attach(GoRouter router) async {
    final initial = await FirebaseMessaging.instance.getInitialMessage();
    if (initial != null) {
      final id = _safeOrderId(initial.data['order_id']);
      if (id != null) router.go('/orders/$id');
    }

    _opened = FirebaseMessaging.onMessageOpenedApp.listen((message) {
      final orderId = message.data['order_id'];
      if (orderId != null) {
        router.go('/orders/$orderId');
      }
      final link = message.data['link'];
      if (link != null) {
        launchUrl(Uri.parse(link), mode: LaunchMode.inAppWebView);
      }
    });

    FirebaseMessaging.onMessage.listen((message) {
      if (kDebugMode) debugPrint('foreground push ${message.messageId}');
    });
  }

  String? _safeOrderId(Object? raw) =>
      raw is String && _orderId.hasMatch(raw) ? raw : null;

  Future<void> detach() async => _opened?.cancel();
}
