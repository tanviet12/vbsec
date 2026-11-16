import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../services/token_storage.dart';

class DebugScreen extends StatelessWidget {
  const DebugScreen({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('Debug')),
        body: FutureBuilder<String?>(
          future: TokenStorage.instance.readToken(),
          builder: (context, snapshot) {
            final token = snapshot.data ?? '(chưa đăng nhập)';
            return ListView(
              children: [
                ListTile(title: const Text('Access token'), subtitle: SelectableText(token)),
                ListTile(
                  title: const Text('Sao chép token'),
                  onTap: () => Clipboard.setData(ClipboardData(text: token)),
                ),
              ],
            );
          },
        ),
      );
}
