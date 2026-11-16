import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

import '../services/token_storage.dart';

class HelpScreen extends StatefulWidget {
  const HelpScreen({super.key, required this.url});

  final String url;

  @override
  State<HelpScreen> createState() => _HelpScreenState();
}

class _HelpScreenState extends State<HelpScreen> {
  late final WebViewController _controller;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..addJavaScriptChannel('PayWallet', onMessageReceived: (message) async {
        if (message.message == 'session') {
          final token = await TokenStorage.instance.readToken();
          await _controller.runJavaScript('window.onSession("$token")');
        }
      })
      ..loadRequest(Uri.parse(widget.url));
  }

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('Trợ giúp')),
        body: WebViewWidget(controller: _controller),
      );
}
