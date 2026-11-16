import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

class AboutScreen extends StatelessWidget {
  const AboutScreen({super.key});

  static final _terms = Uri.https('paywallet.example', '/terms');
  static final _support = Uri.https('paywallet.example', '/support');

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('Về ứng dụng')),
        body: ListView(
          children: [
            ListTile(
              title: const Text('Điều khoản sử dụng'),
              onTap: () => launchUrl(_terms, mode: LaunchMode.externalApplication),
            ),
            ListTile(
              title: const Text('Liên hệ hỗ trợ'),
              onTap: () => launchUrl(_support, mode: LaunchMode.inAppBrowserView),
            ),
          ],
        ),
      );
}
